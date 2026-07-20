//go:build linux

package input

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	evKey      = 0x01
	evSyn      = 0x00
	evMsc      = 0x04
	keyDownVal = 1
	keyUpVal   = 0

	maxKeyCode = 768

	BUS_VIRTUAL = 0x06

	// evdev ioctl: _IOC(_IOC_READ, 'E', 0x20+ev, size)
	// where _IOC_DIRSHIFT=30 _IOC_SIZESHIFT=16 _IOC_TYPESHIFT=8 _IOC_NRSHIFT=0
	eVIOCGBIT = uintptr((2 << 30) | (maxKeyCode/8<<16) | (0x45 << 8) | (0x20 + evKey))
	// EVIOCGID: _IOC(_IOC_READ, 'E', 0x02, sizeof(struct input_id))
	// struct input_id = 4 x uint16 = 8 bytes
	// dir=2 <<30, size=8<<16, type='E'<<8, nr=0x02
	eVIOCGID = uintptr((2 << 30) | (8 << 16) | (0x45 << 8) | 0x02)
)

type evdevInputEvent struct {
	Sec   uint64
	USec  uint64
	Type  uint16
	Code  uint16
	Value int32
}

type inputIDev struct {
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

type evdevCapture struct {
	device string
	rawFd  int
	mu     sync.Mutex
}

func findKeyboardDevice() (string, error) {
	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return "", err
	}
	type candidate struct {
		dev     string
		isReal  bool
		name    string
		bustype uint16
	}
	var candidates []candidate
	for _, dev := range matches {
		fd, err := os.OpenFile(dev, os.O_RDONLY, 0)
		if err != nil {
			continue
		}

		var bits [maxKeyCode / 8]byte
		_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd.Fd(), eVIOCGBIT, uintptr(unsafe.Pointer(&bits[0])), uintptr(len(bits)), 0, 0)
		if errno != 0 {
			fd.Close()
			continue
		}
		if !isKeyboard(bits[:]) {
			fd.Close()
			continue
		}

		// Get bus type to detect virtual devices
		var id inputIDev
		_, _, errno = syscall.Syscall6(syscall.SYS_IOCTL, fd.Fd(), eVIOCGID, uintptr(unsafe.Pointer(&id)), 0, 0, 0)
		name := "unknown"
		if errno == 0 {
			// Get device name via EVIOCGNAME (0x4506)
			var nameBuf [256]byte
			_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), uintptr(0x81004506), uintptr(unsafe.Pointer(&nameBuf[0])))
			if errno == 0 {
				name = string(nameBuf[:])
				if idx := int(0); idx < len(name) && name[idx] != 0 {
					// find null terminator
					for i, c := range name {
						if c == 0 {
							name = name[:i]
							break
						}
					}
				}
			}
		}
		fd.Close()

		isReal := id.Bustype != BUS_VIRTUAL && !containsWord(name, "Virtual")
		candidates = append(candidates, candidate{
			dev:     dev,
			isReal:  isReal,
			name:    name,
			bustype: id.Bustype,
		})
	}

	// Prefer real keyboards, fall back to any keyboard
	var best *candidate
	for i := range candidates {
		c := &candidates[i]
		if best == nil || (c.isReal && !best.isReal) {
			best = c
		}
	}
	if best != nil {
		log.Printf("[capture] evdev using %s (%s) bustype=%d real=%v", best.dev, best.name, best.bustype, best.isReal)
		return best.dev, nil
	}
	return "", errors.New("no keyboard device found")
}

func containsWord(s, word string) bool {
	for i := 0; i <= len(s)-len(word); i++ {
		if s[i:i+len(word)] == word {
			return true
		}
	}
	return false
}

func isKeyboard(bits []byte) bool {
	hasAlpha := false
	hasNum := false
	if len(bits) > 30/8 && (bits[30/8]>>(30%8))&1 != 0 { // KEY_A (30)
		hasAlpha = true
	}
	if len(bits) > 16/8 && (bits[16/8]>>(16%8))&1 != 0 { // KEY_Q (16)
		hasAlpha = true
	}
	if len(bits) > 2/8 && (bits[2/8]>>(2%8))&1 != 0 { // KEY_1 (2)
		hasNum = true
	}
	if len(bits) > 3/8 && (bits[3/8]>>(3%8))&1 != 0 { // KEY_2 (3)
		hasNum = true
	}
	return hasAlpha || hasNum
}

func (e *evdevCapture) init() error {
	e.rawFd = -1
	dev, err := findKeyboardDevice()
	if err != nil {
		return err
	}
	fd, err := syscall.Open(dev, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", dev, err)
	}
	e.device = dev
	e.rawFd = fd
	return nil
}

func (e *evdevCapture) close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.rawFd > 0 {
		syscall.Close(e.rawFd)
		e.rawFd = -1
	}
}

func (e *evdevCapture) run(ch chan<- CaptureResult, stop <-chan struct{}) bool {
	buf := make([]byte, unsafe.Sizeof(evdevInputEvent{}))
	eventSize := int(unsafe.Sizeof(evdevInputEvent{}))

	probationStart := time.Now()
	probationPeriod := 2 * time.Second
	receivedAny := false
	log.Printf("[capture] evdev run loop started on %s", e.device)

	for {
		select {
		case <-stop:
			return true
		default:
		}

		n, err := syscall.Read(e.rawFd, buf)

		if err != nil {
			if !receivedAny && time.Since(probationStart) > probationPeriod {
				log.Printf("[capture] evdev no events after %v, trying next method", probationPeriod)
				return false
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if n == 0 {
			if !receivedAny && time.Since(probationStart) > probationPeriod {
				log.Printf("[capture] evdev no events after %v, trying next method", probationPeriod)
				return false
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if n < eventSize || n%eventSize != 0 {
			continue
		}

		receivedAny = true

		for offset := 0; offset+eventSize <= n; offset += eventSize {
			var ev evdevInputEvent
			ev.Sec = binary.LittleEndian.Uint64(buf[offset : offset+8])
			ev.USec = binary.LittleEndian.Uint64(buf[offset+8 : offset+16])
			ev.Type = binary.LittleEndian.Uint16(buf[offset+16 : offset+18])
			ev.Code = binary.LittleEndian.Uint16(buf[offset+18 : offset+20])
			ev.Value = int32(binary.LittleEndian.Uint32(buf[offset+20 : offset+24]))

			if ev.Type != evKey || ev.Value != keyDownVal {
				continue
			}

			vk := vkFromLinux(ev.Code)
			if vk == 0 {
				continue
			}

			res := CaptureResult{
				Key: KeySpec{
					VK:   vk,
					Scan: ev.Code,
					Ext:  false,
				},
				Mods: nil,
			}

			select {
			case ch <- res:
				return true
			default:
			}
		}
	}
}

func tryEvdevCapture(ch chan<- CaptureResult, stop <-chan struct{}) bool {
	impl := &evdevCapture{}
	if err := impl.init(); err != nil {
		log.Printf("[capture] evdev not available: %v", err)
		return false
	}
	defer impl.close()
	return impl.run(ch, stop)
}
