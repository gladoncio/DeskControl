//go:build linux

package input

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestEvdevCaptureSelf(t *testing.T) {
	impl := &evdevCapture{}
	if err := impl.init(); err != nil {
		t.Fatalf("evdevCapture.init: %v", err)
	}
	defer impl.close()
	t.Logf("evdev opened device: %s", impl.device)

	ch := make(chan CaptureResult, 1)
	stop := make(chan struct{})
	go func() {
		impl.run(ch, stop)
	}()

	t.Logf("PRESS A KEY ON YOUR KEYBOARD within 5 seconds...")
	select {
	case res := <-ch:
		t.Logf("CAPTURED: VK=0x%x Scan=%d", res.Key.VK, res.Key.Scan)
	case <-time.After(5 * time.Second):
		t.Fatal("capture timeout - no key pressed within 5s")
	}
	close(stop)
}

func TestFindKeyboardDevice(t *testing.T) {
	dev, err := findKeyboardDevice()
	if err != nil {
		t.Fatalf("findKeyboardDevice: %v (check permissions: need 'input' group)", err)
	}
	t.Logf("Found keyboard: %s", dev)

	fd, err := os.OpenFile(dev, os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("re-open %s: %v", dev, err)
	}
	defer fd.Close()

	var bits [maxKeyCode / 8]byte
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd.Fd(), eVIOCGBIT, uintptr(unsafe.Pointer(&bits[0])), uintptr(len(bits)), 0, 0)
	if errno != 0 {
		t.Fatalf("ioctl on %s: %v", dev, errno)
	}
	if !isKeyboard(bits[:]) {
		t.Fatalf("%s is not a keyboard according to isKeyboard()", dev)
	}
	t.Logf("isKeyboard() confirmed for %s", dev)
}

func TestEvdevReadEvents(t *testing.T) {
	dev, err := findKeyboardDevice()
	if err != nil {
		t.Fatalf("findKeyboardDevice: %v", err)
	}

	fd, err := os.OpenFile(dev, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		t.Fatalf("open %s: %v", dev, err)
	}
	defer fd.Close()

	eventSize := int(unsafe.Sizeof(evdevInputEvent{}))
	buf := make([]byte, eventSize)
	t.Logf("Event size: %d bytes", eventSize)
	t.Logf("PRESS A KEY within 5 seconds...")

	deadline := time.Now().Add(5 * time.Second)
	count := 0
	for time.Now().Before(deadline) {
		n, err := fd.Read(buf)
		if err != nil {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		if n < eventSize || n%eventSize != 0 {
			continue
		}
		for offset := 0; offset+eventSize <= n; offset += eventSize {
			evType := binary.LittleEndian.Uint16(buf[offset+16 : offset+18])
			evCode := binary.LittleEndian.Uint16(buf[offset+18 : offset+20])
			evValue := int32(binary.LittleEndian.Uint32(buf[offset+20 : offset+24]))
			count++

			if evType == evKey {
				action := map[int32]string{0: "UP", 1: "DOWN", 2: "HOLD"}[evValue]
				if action == "" {
					action = fmt.Sprintf("?%d", evValue)
				}
				t.Logf("KEY %s code=%d", action, evCode)
				if evValue == 1 {
					vk := vkFromLinux(evCode)
					t.Logf("  → VK=0x%x (%d)", vk, vk)
				}
			}
		}
	}
	t.Logf("Total events read: %d", count)
	if count == 0 {
		t.Fatal("No events received - device is not producing events")
	}
}
