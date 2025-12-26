//go:build windows

package input

import (
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	INPUT_MOUSE    = 0
	INPUT_KEYBOARD = 1

	// Mouse flags
	MOUSEEVENTF_MOVE      = 0x0001
	MOUSEEVENTF_LEFTDOWN  = 0x0002
	MOUSEEVENTF_LEFTUP    = 0x0004
	MOUSEEVENTF_RIGHTDOWN = 0x0008
	MOUSEEVENTF_RIGHTUP   = 0x0010
	MOUSEEVENTF_WHEEL     = 0x0800

	// Keyboard flags
	KEYEVENTF_EXTENDEDKEY = 0x0001
	KEYEVENTF_KEYUP       = 0x0002
	KEYEVENTF_UNICODE     = 0x0004
)

type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// INPUT struct con "union" como blob de 32 bytes (compatible amd64)
type INPUT struct {
	Type uint32
	_    uint32
	Data [32]byte
}

var (
	user32    = syscall.NewLazyDLL("user32.dll")
	sendInput = user32.NewProc("SendInput")
)

type WindowsInput struct{}

func New() *WindowsInput { return &WindowsInput{} }

func (w *WindowsInput) send(inputs []INPUT) error {
	if len(inputs) == 0 {
		return nil
	}
	r1, _, err := sendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func mouseInput(flags uint32, dx, dy int32, mouseData uint32) INPUT {
	var in INPUT
	in.Type = INPUT_MOUSE
	mi := (*MOUSEINPUT)(unsafe.Pointer(&in.Data[0]))
	mi.Dx = dx
	mi.Dy = dy
	mi.MouseData = mouseData
	mi.DwFlags = flags
	return in
}

func isExtendedVK(vk uint16) bool {
	switch vk {
	case 0x2E: // VK_DELETE
		return true
	case 0x25, 0x26, 0x27, 0x28: // arrows
		return true
	case 0x2D, 0x24, 0x23, 0x21, 0x22: // ins, home, end, pgup, pgdn
		return true
	default:
		return false
	}
}

func keyInput(vk uint16, scan uint16, flags uint32) INPUT {
	var in INPUT
	in.Type = INPUT_KEYBOARD
	ki := (*KEYBDINPUT)(unsafe.Pointer(&in.Data[0]))
	ki.WVk = vk
	ki.WScan = scan

	if vk != 0 && isExtendedVK(vk) {
		flags |= KEYEVENTF_EXTENDEDKEY
	}
	ki.DwFlags = flags
	return in
}

func keyInputSpec(k KeySpec, flags uint32) INPUT {
	if k.Ext {
		flags |= KEYEVENTF_EXTENDEDKEY
	}
	return keyInput(k.VK, k.Scan, flags)
}

func (w *WindowsInput) MoveMouse(dx, dy int32) error {
	return w.send([]INPUT{mouseInput(MOUSEEVENTF_MOVE, dx, dy, 0)})
}

func (w *WindowsInput) MouseClick(button string) error {
	button = strings.ToLower(button)
	var down, up uint32
	switch button {
	case "right":
		down, up = MOUSEEVENTF_RIGHTDOWN, MOUSEEVENTF_RIGHTUP
	default:
		down, up = MOUSEEVENTF_LEFTDOWN, MOUSEEVENTF_LEFTUP
	}
	return w.send([]INPUT{
		mouseInput(down, 0, 0, 0),
		mouseInput(up, 0, 0, 0),
	})
}

func (w *WindowsInput) MouseDown(button string) error {
	button = strings.ToLower(button)
	var flag uint32
	switch button {
	case "right":
		flag = MOUSEEVENTF_RIGHTDOWN
	default:
		flag = MOUSEEVENTF_LEFTDOWN
	}
	return w.send([]INPUT{mouseInput(flag, 0, 0, 0)})
}

func (w *WindowsInput) MouseUp(button string) error {
	button = strings.ToLower(button)
	var flag uint32
	switch button {
	case "right":
		flag = MOUSEEVENTF_RIGHTUP
	default:
		flag = MOUSEEVENTF_LEFTUP
	}
	return w.send([]INPUT{mouseInput(flag, 0, 0, 0)})
}

func (w *WindowsInput) MouseScroll(dy int32) error {
	return w.send([]INPUT{mouseInput(MOUSEEVENTF_WHEEL, 0, 0, uint32(int32(dy)))})
}

func (w *WindowsInput) KeyText(text string) error {
	if text == "" {
		return nil
	}
	u := utf16.Encode([]rune(text))
	inputs := make([]INPUT, 0, len(u)*2)

	for _, cu := range u {
		inputs = append(inputs,
			keyInput(0, cu, KEYEVENTF_UNICODE),
			keyInput(0, cu, KEYEVENTF_UNICODE|KEYEVENTF_KEYUP),
		)
	}
	return w.send(inputs)
}

func (w *WindowsInput) Key(key string) error {
	vk := vkFromKey(strings.ToLower(key))
	if vk == 0 {
		return nil
	}
	return w.send([]INPUT{
		keyInput(vk, 0, 0),
		keyInput(vk, 0, KEYEVENTF_KEYUP),
	})
}

func (w *WindowsInput) KeyDown(key string) error {
	vk := vkFromKey(strings.ToLower(key))
	if vk == 0 {
		return nil
	}
	return w.send([]INPUT{keyInput(vk, 0, 0)})
}

func (w *WindowsInput) KeyUp(key string) error {
	vk := vkFromKey(strings.ToLower(key))
	if vk == 0 {
		return nil
	}
	return w.send([]INPUT{keyInput(vk, 0, KEYEVENTF_KEYUP)})
}

func (w *WindowsInput) Hotkey(mods []string, key string) error {
	var inputs []INPUT

	for _, m := range mods {
		vk := vkFromMod(strings.ToLower(m))
		if vk != 0 {
			inputs = append(inputs, keyInput(vk, 0, 0))
		}
	}

	main := vkFromKey(strings.ToLower(key))
	if main != 0 {
		inputs = append(inputs,
			keyInput(main, 0, 0),
			keyInput(main, 0, KEYEVENTF_KEYUP),
		)
	}

	for i := len(mods) - 1; i >= 0; i-- {
		vk := vkFromMod(strings.ToLower(mods[i]))
		if vk != 0 {
			inputs = append(inputs, keyInput(vk, 0, KEYEVENTF_KEYUP))
		}
	}

	return w.send(inputs)
}

// ---- Stable VK/Scan execution (phone-defined bindings) ----

func (w *WindowsInput) KeyVK(k KeySpec) error {
	if k.VK == 0 && k.Scan == 0 {
		return nil
	}
	return w.send([]INPUT{keyInputSpec(k, 0), keyInputSpec(k, KEYEVENTF_KEYUP)})
}

func (w *WindowsInput) KeyDownVK(k KeySpec) error {
	if k.VK == 0 && k.Scan == 0 {
		return nil
	}
	return w.send([]INPUT{keyInputSpec(k, 0)})
}

func (w *WindowsInput) KeyUpVK(k KeySpec) error {
	if k.VK == 0 && k.Scan == 0 {
		return nil
	}
	return w.send([]INPUT{keyInputSpec(k, KEYEVENTF_KEYUP)})
}

func (w *WindowsInput) HotkeyVK(mods []string, k KeySpec) error {
	var inputs []INPUT
	for _, m := range mods {
		vk := vkFromMod(strings.ToLower(m))
		if vk != 0 {
			inputs = append(inputs, keyInput(vk, 0, 0))
		}
	}
	if k.VK != 0 || k.Scan != 0 {
		inputs = append(inputs,
			keyInputSpec(k, 0),
			keyInputSpec(k, KEYEVENTF_KEYUP),
		)
	}
	for i := len(mods) - 1; i >= 0; i-- {
		vk := vkFromMod(strings.ToLower(mods[i]))
		if vk != 0 {
			inputs = append(inputs, keyInput(vk, 0, KEYEVENTF_KEYUP))
		}
	}
	return w.send(inputs)
}

func vkFromMod(m string) uint16 {
	switch m {
	case "ctrl", "control":
		return 0x11 // VK_CONTROL
	case "alt":
		return 0x12 // VK_MENU
	case "shift":
		return 0x10 // VK_SHIFT
	case "win", "windows", "meta":
		return 0x5B // VK_LWIN
	default:
		return 0
	}
}

func vkFromKey(k string) uint16 {
	switch k {
	case "enter":
		return 0x0D
	case "backspace":
		return 0x08
	case "tab":
		return 0x09
	case "esc", "escape":
		return 0x1B
	case "space":
		return 0x20

	case "up":
		return 0x26
	case "down":
		return 0x28
	case "left":
		return 0x25
	case "right":
		return 0x27

	case "delete", "del":
		return 0x2E // VK_DELETE
	case "win", "windows":
		return 0x5B // VK_LWIN

	// Audio / media
	case "vol_mute":
		return 0xAD // VK_VOLUME_MUTE
	case "vol_down":
		return 0xAE // VK_VOLUME_DOWN
	case "vol_up":
		return 0xAF // VK_VOLUME_UP
	case "media_next":
		return 0xB0 // VK_MEDIA_NEXT_TRACK
	case "media_prev":
		return 0xB1 // VK_MEDIA_PREV_TRACK
	case "media_play_pause":
		return 0xB3 // VK_MEDIA_PLAY_PAUSE
	}

	// letters
	if len(k) == 1 {
		c := k[0]
		if c >= 'a' && c <= 'z' {
			return uint16(c - 32) // 'A'
		}
		if c >= '0' && c <= '9' {
			return uint16(c)
		}
	}

	// F1..F12
	if strings.HasPrefix(k, "f") && len(k) <= 3 {
		n := 0
		for _, ch := range k[1:] {
			if ch < '0' || ch > '9' {
				n = 0
				break
			}
			n = n*10 + int(ch-'0')
		}
		if n >= 1 && n <= 12 {
			return uint16(0x70 + (n - 1))
		}
	}

	return 0
}
