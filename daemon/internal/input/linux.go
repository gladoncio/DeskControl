//go:build linux

package input

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

type linuxInput struct {
	uinput *uinputDriver
	xtest  *xtestDriver
}

func trySpawnUinput() *uinputDriver {
	d, err := newUinputDriver()
	if err != nil {
		log.Printf("[input] uinput not available: %v", err)
		return nil
	}
	log.Printf("[input] using uinput driver")
	return d
}

func trySpawnXTest() *xtestDriver {
	d, err := newXTestDriver()
	if err != nil {
		log.Printf("[input] XTest not available: %v", err)
		return nil
	}
	log.Printf("[input] using XTest driver")
	return d
}

func New() *linuxInput {
	d := &linuxInput{}
	if u := trySpawnUinput(); u != nil {
		d.uinput = u
	}
	if x := trySpawnXTest(); x != nil {
		d.xtest = x
	}
	if d.uinput == nil && d.xtest == nil {
		log.Printf("[input] WARNING: no input driver available (uinput nor XTest)")
	}
	return d
}

func (d *linuxInput) MoveMouse(dx, dy int32) error {
	if d.uinput != nil {
		return d.uinput.MoveMouse(dx, dy)
	}
	if d.xtest != nil {
		return d.xtest.MoveMouse(dx, dy)
	}
	return errNoDriver
}

func (d *linuxInput) MouseClick(button string) error {
	if d.uinput != nil {
		return d.uinput.MouseClick(button)
	}
	if d.xtest != nil {
		return d.xtest.MouseClick(button)
	}
	return errNoDriver
}

func (d *linuxInput) MouseDown(button string) error {
	if d.uinput != nil {
		return d.uinput.MouseDown(button)
	}
	if d.xtest != nil {
		return d.xtest.MouseDown(button)
	}
	return errNoDriver
}

func (d *linuxInput) MouseUp(button string) error {
	if d.uinput != nil {
		return d.uinput.MouseUp(button)
	}
	if d.xtest != nil {
		return d.xtest.MouseUp(button)
	}
	return errNoDriver
}

func (d *linuxInput) MouseScroll(dy int32) error {
	if d.uinput != nil {
		return d.uinput.MouseScroll(dy)
	}
	if d.xtest != nil {
		return d.xtest.MouseScroll(dy)
	}
	return errNoDriver
}

func (d *linuxInput) KeyText(text string) error {
	// Prefer XTest for text input — it uses X11 keysym lookup which respects
	// the active keyboard layout. uinput sends raw keycodes (US QWERTY positions)
	// that may produce wrong characters on non-US layouts.
	if d.xtest != nil {
		return d.xtest.KeyText(text)
	}
	if d.uinput != nil {
		return d.uinput.KeyText(text)
	}
	return errNoDriver
}

func (d *linuxInput) Key(key string) error {
	if d.uinput != nil {
		return d.uinput.Key(key)
	}
	if d.xtest != nil {
		return d.xtest.Key(key)
	}
	return errNoDriver
}

func (d *linuxInput) KeyDown(key string) error {
	if d.uinput != nil {
		return d.uinput.KeyDown(key)
	}
	if d.xtest != nil {
		return d.xtest.KeyDown(key)
	}
	return errNoDriver
}

func (d *linuxInput) KeyUp(key string) error {
	if d.uinput != nil {
		return d.uinput.KeyUp(key)
	}
	if d.xtest != nil {
		return d.xtest.KeyUp(key)
	}
	return errNoDriver
}

func (d *linuxInput) Hotkey(mods []string, key string) error {
	if d.uinput != nil {
		return d.uinput.Hotkey(mods, key)
	}
	if d.xtest != nil {
		return d.xtest.Hotkey(mods, key)
	}
	return errNoDriver
}

func (d *linuxInput) KeyVK(k KeySpec) error {
	if d.uinput != nil {
		return d.uinput.KeyVK(k)
	}
	if d.xtest != nil {
		return d.xtest.KeyVK(k)
	}
	return errNoDriver
}

func (d *linuxInput) KeyDownVK(k KeySpec) error {
	if d.uinput != nil {
		return d.uinput.KeyDownVK(k)
	}
	if d.xtest != nil {
		return d.xtest.KeyDownVK(k)
	}
	return errNoDriver
}

func (d *linuxInput) KeyUpVK(k KeySpec) error {
	if d.uinput != nil {
		return d.uinput.KeyUpVK(k)
	}
	if d.xtest != nil {
		return d.xtest.KeyUpVK(k)
	}
	return errNoDriver
}

func (d *linuxInput) HotkeyVK(mods []string, k KeySpec) error {
	if d.uinput != nil {
		return d.uinput.HotkeyVK(mods, k)
	}
	if d.xtest != nil {
		return d.xtest.HotkeyVK(mods, k)
	}
	return errNoDriver
}

func (d *linuxInput) CaptureNextKey(timeoutMs int) (CaptureResult, error) {
	return captureNextKey(timeoutMs)
}

func (d *linuxInput) ListApps() ([]AppInfo, error) {
	return listApps()
}

func (d *linuxInput) AppAction(hwnd uintptr, action string) error {
	return appAction(hwnd, action)
}

func (d *linuxInput) DriverInfo() string {
	session := os.Getenv("XDG_SESSION_TYPE")
	if session == "" {
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			session = "wayland"
		} else if os.Getenv("DISPLAY") != "" {
			session = "x11"
		} else {
			session = "unknown"
		}
	}
	parts := []string{"Linux", session}
	if d.xtest != nil {
		parts = append(parts, "xtest(text)")
	}
	if d.uinput != nil {
		parts = append(parts, "uinput(key)")
	}
	if len(parts) == 2 {
		parts = append(parts, "none")
	}
	return strings.Join(parts, " · ")
}

// vkFromKey maps key names to Windows VK codes (Linux build)
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
		return 0x2E
	case "win", "windows":
		return 0x5B
	case "vol_mute":
		return 0xAD
	case "vol_down":
		return 0xAE
	case "vol_up":
		return 0xAF
	case "media_next":
		return 0xB0
	case "media_prev":
		return 0xB1
	case "media_play_pause":
		return 0xB3
	}

	if len(k) == 1 {
		c := k[0]
		if c >= 'a' && c <= 'z' {
			return uint16(c - 32)
		}
		if c >= '0' && c <= '9' {
			return uint16(c)
		}
	}

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

func (d *linuxInput) ClipboardGetText() (string, error) {
	// Try xclip (X11), xsel (X11), then wl-paste (Wayland)
	if out, err := exec.Command("xclip", "-o", "-selection", "clipboard").Output(); err == nil {
		return strings.TrimRight(string(out), "\n\r"), nil
	}
	if out, err := exec.Command("xsel", "-o", "-b").Output(); err == nil {
		return strings.TrimRight(string(out), "\n\r"), nil
	}
	if out, err := exec.Command("wl-paste").Output(); err == nil {
		return strings.TrimRight(string(out), "\n\r"), nil
	}
	return "", exec.ErrNotFound
}

func (d *linuxInput) ClipboardSetText(text string) error {
	// Try xclip (X11), xsel (X11), then wl-copy (Wayland)
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err == nil {
		return nil
	}
	cmd = exec.Command("xsel", "-i", "-b")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err == nil {
		return nil
	}
	cmd = exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func vkFromMod(m string) uint16 {
	switch m {
	case "ctrl", "control":
		return 0x11
	case "alt":
		return 0x12
	case "shift":
		return 0x10
	case "win", "windows", "meta":
		return 0x5B
	default:
		return 0
	}
}
