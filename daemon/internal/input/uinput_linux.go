//go:build linux

package input

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	uinputDeviceName = "DeskControl Virtual Input"

	EV_KEY = 0x01
	EV_REL = 0x02
	EV_SYN = 0x00
	EV_ABS = 0x03

	REL_X = 0x00
	REL_Y = 0x01
	REL_WHEEL = 0x08

	BTN_LEFT   = 0x110
	BTN_RIGHT  = 0x111
	BTN_MIDDLE = 0x112

	UI_SET_EVBIT  = 0x40045564
	UI_SET_KEYBIT = 0x40045565
	UI_SET_RELBIT = 0x40045566
	UI_SET_ABSBIT = 0x40045567
	UI_DEV_SETUP  = 0x405c5503
	UI_DEV_CREATE = 0x5501
	UI_DEV_DESTROY = 0x5502

	BUS_USB = 0x03

	SYN_REPORT = 0x00
)

type inputID struct {
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

type uinputSetup struct {
	Name       [80]byte
	ID         inputID
	EffectsMax uint32
}

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type uinputDriver struct {
	mu          sync.Mutex
	fd          *os.File
	relInitDone bool
}

var errNoDriver = errors.New("no input driver available")

// vkToLinux maps Windows VK codes to Linux input event codes
func vkToLinux(vk uint16) uint16 {
	switch vk {
	case 0x08:
		return 14 // KEY_BACKSPACE
	case 0x09:
		return 15 // KEY_TAB
	case 0x0D:
		return 28 // KEY_ENTER
	case 0x10:
		return 42 // KEY_LEFTSHIFT
	case 0x11:
		return 29 // KEY_LEFTCTRL
	case 0x12:
		return 56 // KEY_LEFTALT
	case 0x1B:
		return 1 // KEY_ESC
	case 0x20:
		return 57 // KEY_SPACE
	case 0x21:
		return 104 // KEY_PAGEUP
	case 0x22:
		return 109 // KEY_PAGEDOWN
	case 0x23:
		return 107 // KEY_END
	case 0x24:
		return 102 // KEY_HOME
	case 0x25:
		return 105 // KEY_LEFT
	case 0x26:
		return 103 // KEY_UP
	case 0x27:
		return 106 // KEY_RIGHT
	case 0x28:
		return 108 // KEY_DOWN
	case 0x2D:
		return 110 // KEY_INSERT
	case 0x2E:
		return 111 // KEY_DELETE
	case 0x5B:
		return 125 // KEY_LEFTMETA
	case 0x5C:
		return 126 // KEY_RIGHTMETA
	case 0xAD:
		return 113 // KEY_MUTE
	case 0xAE:
		return 114 // KEY_VOLUMEDOWN
	case 0xAF:
		return 115 // KEY_VOLUMEUP
	case 0xB0:
		return 163 // KEY_NEXTSONG
	case 0xB1:
		return 165 // KEY_PREVIOUSSONG
	case 0xB3:
		return 164 // KEY_PLAYPAUSE
	case 0x70:
		return 59 // KEY_F1
	case 0x71:
		return 60 // KEY_F2
	case 0x72:
		return 61 // KEY_F3
	case 0x73:
		return 62 // KEY_F4
	case 0x74:
		return 63 // KEY_F5
	case 0x75:
		return 64 // KEY_F6
	case 0x76:
		return 65 // KEY_F7
	case 0x77:
		return 66 // KEY_F8
	case 0x78:
		return 67 // KEY_F9
	case 0x79:
		return 68 // KEY_F10
	case 0x7A:
		return 87 // KEY_F11
	case 0x7B:
		return 88 // KEY_F12
	}

	if vk >= 'A' && vk <= 'Z' {
		return usQwertyLetter(uint8(vk))
	}
	if vk >= '0' && vk <= '9' {
		return digitToLinux[vk-'0']
	}

	return 0
}

func vkFromLinux(lc uint16) uint16 {
	for vk, code := range vkToLinuxMap {
		if code == lc {
			return vk
		}
	}
	return 0
}

var vkToLinuxMap = func() map[uint16]uint16 {
	m := make(map[uint16]uint16)
	for vk := uint16(0x08); vk <= 0x7B; vk++ {
		if lc := vkToLinux(vk); lc != 0 {
			m[vk] = lc
		}
	}
	for c := 'A'; c <= 'Z'; c++ {
		m[uint16(c)] = uint16(c - 'A' + 30)
	}
	for c := '0'; c <= '9'; c++ {
		m[uint16(c)] = uint16(c - '0' + 2)
	}
	return m
}()

func keyNameToLinux(key string) uint16 {
	vk := vkFromKey(strings.ToLower(key))
	if vk == 0 {
		return 0
	}
	return vkToLinux(vk)
}

const uinputPath = "/dev/uinput"

func newUinputDriver() (*uinputDriver, error) {
	f, err := os.OpenFile(uinputPath, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", uinputPath, err)
	}

	d := &uinputDriver{fd: f}
	if err := d.init(); err != nil {
		f.Close()
		return nil, err
	}
	return d, nil
}

func (d *uinputDriver) ioctl(cmd uintptr, val int) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, d.fd.Fd(), cmd, uintptr(val))
	if err != 0 && err != syscall.ENOTTY {
		return err
	}
	return nil
}

func (d *uinputDriver) writeEvent(typ, code uint16, value int32) error {
	ev := inputEvent{
		Type:  typ,
		Code:  code,
		Value: value,
	}
	return binary.Write(d.fd, binary.LittleEndian, &ev)
}

func (d *uinputDriver) syn() error {
	return d.writeEvent(EV_SYN, SYN_REPORT, 0)
}

func (d *uinputDriver) init() error {
	if err := d.ioctl(UI_SET_EVBIT, EV_KEY); err != nil {
		return fmt.Errorf("set EV_KEY: %w", err)
	}
	if err := d.ioctl(UI_SET_EVBIT, EV_REL); err != nil {
		return fmt.Errorf("set EV_REL: %w", err)
	}

	mouseBtns := []int{BTN_LEFT, BTN_RIGHT, BTN_MIDDLE}
	for _, b := range mouseBtns {
		if err := d.ioctl(UI_SET_KEYBIT, b); err != nil {
			return fmt.Errorf("set btn keybit %d: %w", b, err)
		}
	}

	relAxes := []int{REL_X, REL_Y, REL_WHEEL}
	for _, a := range relAxes {
		if err := d.ioctl(UI_SET_RELBIT, a); err != nil {
			return fmt.Errorf("set relbit %d: %w", a, err)
		}
	}

	for _, v := range vkToLinuxMap {
		if err := d.ioctl(UI_SET_KEYBIT, int(v)); err != nil {
			return fmt.Errorf("set keybit %d: %w", v, err)
		}
	}

	setup := uinputSetup{
		ID: inputID{
			Bustype: BUS_USB,
			Vendor:  0x1234,
			Product: 0x5678,
			Version: 1,
		},
	}
	copy(setup.Name[:], uinputDeviceName)

	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, d.fd.Fd(), UI_DEV_SETUP, uintptr(unsafe.Pointer(&setup))); err != 0 {
		d.fd.Close()
		return fmt.Errorf("UI_DEV_SETUP: %w", err)
	}
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, d.fd.Fd(), UI_DEV_CREATE, 0); err != 0 {
		d.fd.Close()
		return fmt.Errorf("UI_DEV_CREATE: %w", err)
	}

	return nil
}

func (d *uinputDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fd == nil {
		return nil
	}
	syscall.Syscall(syscall.SYS_IOCTL, d.fd.Fd(), UI_DEV_DESTROY, 0)
	return d.fd.Close()
}

func (d *uinputDriver) MoveMouse(dx, dy int32) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writeEvent(EV_REL, REL_X, dx); err != nil {
		return err
	}
	if err := d.writeEvent(EV_REL, REL_Y, dy); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) MouseClick(button string) error {
	code := mouseBtnCode(button)
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writeEvent(EV_KEY, code, 1); err != nil {
		return err
	}
	if err := d.syn(); err != nil {
		return err
	}
	if err := d.writeEvent(EV_KEY, code, 0); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) MouseDown(button string) error {
	code := mouseBtnCode(button)
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writeEvent(EV_KEY, code, 1); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) MouseUp(button string) error {
	code := mouseBtnCode(button)
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writeEvent(EV_KEY, code, 0); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) MouseScroll(dy int32) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writeEvent(EV_REL, REL_WHEEL, -dy); err != nil {
		return err
	}
	return d.syn()
}

func charToLinuxKeycode(c rune) uint16 {
	// Linux input event codes for letters (US QWERTY positions)
	switch {
	case c >= 'a' && c <= 'z':
		return usQwertyLetter(uint8(c - 'a' + 'A'))
	case c >= 'A' && c <= 'Z':
		return usQwertyLetter(uint8(c))
	default:
		return 0
	}
}

func usQwertyLetter(letter uint8) uint16 {
	switch letter {
	case 'A':
		return 30
	case 'B':
		return 48
	case 'C':
		return 46
	case 'D':
		return 32
	case 'E':
		return 18
	case 'F':
		return 33
	case 'G':
		return 34
	case 'H':
		return 35
	case 'I':
		return 23
	case 'J':
		return 36
	case 'K':
		return 37
	case 'L':
		return 38
	case 'M':
		return 50
	case 'N':
		return 49
	case 'O':
		return 24
	case 'P':
		return 25
	case 'Q':
		return 16
	case 'R':
		return 19
	case 'S':
		return 31
	case 'T':
		return 20
	case 'U':
		return 22
	case 'V':
		return 47
	case 'W':
		return 17
	case 'X':
		return 45
	case 'Y':
		return 21
	case 'Z':
		return 44
	}
	return 0
}

var digitToLinux = [...]uint16{11, 2, 3, 4, 5, 6, 7, 8, 9, 10} // 0→11, 1→2, ...

func (d *uinputDriver) KeyText(text string) error {
	if text == "" {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, r := range text {
		var code uint16
		var shift bool

		code = charToLinuxKeycode(r)
		if code == 0 {
			if r >= '0' && r <= '9' {
				code = digitToLinux[r-'0']
			} else {
				code = charToLinux(r)
				if code == 0 {
					continue
				}
			}
		} else if r >= 'A' && r <= 'Z' {
			shift = true
		}

		if shift {
			d.writeEvent(EV_KEY, 42, 1)
		}
		d.writeEvent(EV_KEY, code, 1)
		d.syn()
		d.writeEvent(EV_KEY, code, 0)
		if shift {
			d.writeEvent(EV_KEY, 42, 0)
		}
		d.syn()
	}
	return nil
}

func charToLinux(r rune) uint16 {
	switch r {
	case ' ':
		return 57
	case '\n':
		return 28
	case '\t':
		return 15
	case '.':
		return 52
	case ',':
		return 51
	case '-':
		return 12
	case '=':
		return 13
	case '+':
		return 13
	case '/':
		return 53
	case '*':
		return 55
	case ';':
		return 39
	case '\'':
		return 40
	case '[':
		return 26
	case ']':
		return 27
	case '\\':
		return 43
	case '`':
		return 41
	}
	return 0
}

func (d *uinputDriver) Key(key string) error {
	code := keyNameToLinux(key)
	if code == 0 {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writeEvent(EV_KEY, code, 1); err != nil {
		return err
	}
	if err := d.syn(); err != nil {
		return err
	}
	if err := d.writeEvent(EV_KEY, code, 0); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) KeyDown(key string) error {
	code := keyNameToLinux(key)
	if code == 0 {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writeEvent(EV_KEY, code, 1); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) KeyUp(key string) error {
	code := keyNameToLinux(key)
	if code == 0 {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writeEvent(EV_KEY, code, 0); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) Hotkey(mods []string, key string) error {
	codes := make([]uint16, 0, len(mods)+1)
	for _, m := range mods {
		c := modNameToLinux(m)
		if c != 0 {
			codes = append(codes, c)
		}
	}
	mainCode := keyNameToLinux(key)
	if mainCode != 0 {
		codes = append(codes, mainCode)
	}
	if len(codes) == 0 {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range codes {
		d.writeEvent(EV_KEY, c, 1)
	}
	d.syn()
	for i := len(codes) - 1; i >= 0; i-- {
		d.writeEvent(EV_KEY, codes[i], 0)
	}
	return d.syn()
}

func modNameToLinux(m string) uint16 {
	switch strings.ToLower(m) {
	case "ctrl", "control":
		return 29 // KEY_LEFTCTRL
	case "alt":
		return 56 // KEY_LEFTALT
	case "shift":
		return 42 // KEY_LEFTSHIFT
	case "win", "windows", "meta":
		return 125 // KEY_LEFTMETA
	default:
		return 0
	}
}

func (d *uinputDriver) pressVK(k KeySpec, state int32) error {
	code := vkToLinux(k.VK)
	if code == 0 && k.Scan != 0 {
		code = k.Scan
	}
	if code == 0 {
		return nil
	}
	return d.writeEvent(EV_KEY, code, state)
}

func (d *uinputDriver) KeyVK(k KeySpec) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.pressVK(k, 1); err != nil {
		return err
	}
	if err := d.syn(); err != nil {
		return err
	}
	if err := d.pressVK(k, 0); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) KeyDownVK(k KeySpec) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.pressVK(k, 1); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) KeyUpVK(k KeySpec) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.pressVK(k, 0); err != nil {
		return err
	}
	return d.syn()
}

func (d *uinputDriver) HotkeyVK(mods []string, k KeySpec) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, m := range mods {
		c := modNameToLinux(m)
		if c != 0 {
			d.writeEvent(EV_KEY, c, 1)
		}
	}
	code := vkToLinux(k.VK)
	if code == 0 && k.Scan != 0 {
		code = k.Scan
	}
	if code != 0 {
		d.writeEvent(EV_KEY, code, 1)
		d.writeEvent(EV_KEY, code, 0)
	}
	for i := len(mods) - 1; i >= 0; i-- {
		c := modNameToLinux(mods[i])
		if c != 0 {
			d.writeEvent(EV_KEY, c, 0)
		}
	}
	return d.syn()
}

func mouseBtnCode(button string) uint16 {
	switch strings.ToLower(button) {
	case "right":
		return BTN_RIGHT
	case "middle":
		return BTN_MIDDLE
	default:
		return BTN_LEFT
	}
}
