//go:build linux

package input

/*
#cgo LDFLAGS: -lX11 -lXtst
#include <X11/Xlib.h>
#include <X11/extensions/XTest.h>
#include <X11/keysym.h>
#include <X11/keysymdef.h>
#include <stdlib.h>
#include <string.h>

#ifndef XF86XK_AudioMute
#define XF86XK_AudioMute 0x1008FF13
#define XF86XK_AudioLowerVolume 0x1008FF11
#define XF86XK_AudioRaiseVolume 0x1008FF12
#define XF86XK_AudioNext 0x1008FF17
#define XF86XK_AudioPrev 0x1008FF16
#define XF86XK_AudioPlay 0x1008FF14
#endif

static inline void xFakeKey(Display *d, KeyCode kc, int press) {
    XTestFakeKeyEvent(d, (unsigned int)kc, press, CurrentTime);
}

// Find keycode + modifiers for a target keysym using full keyboard mapping.
// Returns keycode (0 if not found) and sets shift/altgr flags.
static KeyCode findKeycode(Display *d, KeySym target, int *shift, int *altgr) {
    *shift = 0;
    *altgr = 0;
    if (target == NoSymbol) return 0;

    int min_kc, max_kc;
    XDisplayKeycodes(d, &min_kc, &max_kc);
    int keysyms_per_kc;
    KeySym *syms = XGetKeyboardMapping(d, min_kc, max_kc - min_kc + 1, &keysyms_per_kc);
    if (!syms) return 0;

    KeyCode result = 0;
    for (int kc = min_kc; kc <= max_kc; kc++) {
        int idx = (kc - min_kc) * keysyms_per_kc;
        // Only check groups 0-3 (unshifted, shifted, AltGr, AltGr+Shift).
        // Groups 4+ require non-standard modifiers (Level3, Level5, etc.)
        // that we cannot reliably simulate via XTest.
        int max_g = keysyms_per_kc;
        if (max_g > 4) max_g = 4;
        for (int g = 0; g < max_g; g++) {
            if (syms[idx + g] == target) {
                result = kc;
                if (g % 2 == 1) *shift = 1;
                if (g / 2 == 1) *altgr = 1;
                goto found;
            }
        }
    }
found:
    XFree(syms);
    return result;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

type xtestDriver struct {
	mu          sync.Mutex
	display     *C.Display
	clipWindow  C.Window
	clipAtom    C.Atom
	targetsAtom C.Atom
	utf8Atom    C.Atom
	textAtom    C.Atom
	clipboard   string
}

func newXTestDriver() (*xtestDriver, error) {
	d := C.XOpenDisplay(nil)
	if d == nil {
		return nil, errors.New("cannot open X display")
	}
	return &xtestDriver{display: d}, nil
}

func (x *xtestDriver) Close() {
	if x.display != nil {
		C.XCloseDisplay(x.display)
	}
}

func (x *xtestDriver) MoveMouse(dx, dy int32) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	var root, child C.Window
	var rx, ry, wx, wy C.int
	var mask C.uint
	C.XQueryPointer(x.display, C.XDefaultRootWindow(x.display), &root, &child, &rx, &ry, &wx, &wy, &mask)
	C.XTestFakeMotionEvent(x.display, -1, rx+C.int(dx), ry+C.int(dy), C.CurrentTime)
	C.XFlush(x.display)
	return nil
}

func (x *xtestDriver) MouseClick(button string) error {
	btn := C.uint(mouseXButton(button))
	x.mu.Lock()
	defer x.mu.Unlock()
	C.XTestFakeButtonEvent(x.display, btn, 1, C.CurrentTime)
	C.XFlush(x.display)
	C.XTestFakeButtonEvent(x.display, btn, 0, C.CurrentTime)
	C.XFlush(x.display)
	return nil
}

func (x *xtestDriver) MouseDown(button string) error {
	btn := C.uint(mouseXButton(button))
	x.mu.Lock()
	defer x.mu.Unlock()
	C.XTestFakeButtonEvent(x.display, btn, 1, C.CurrentTime)
	C.XFlush(x.display)
	return nil
}

func (x *xtestDriver) MouseUp(button string) error {
	btn := C.uint(mouseXButton(button))
	x.mu.Lock()
	defer x.mu.Unlock()
	C.XTestFakeButtonEvent(x.display, btn, 0, C.CurrentTime)
	C.XFlush(x.display)
	return nil
}

func (x *xtestDriver) MouseScroll(dy int32) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	clicks := dy / 120
	if clicks == 0 {
		if dy > 0 {
			clicks = 1
		} else {
			clicks = -1
		}
	}
	btn := C.uint(4)
	if clicks < 0 {
		btn = 5
		clicks = -clicks
	}
	for i := int32(0); i < clicks; i++ {
		C.XTestFakeButtonEvent(x.display, btn, 1, C.CurrentTime)
		C.XTestFakeButtonEvent(x.display, btn, 0, C.CurrentTime)
	}
	C.XFlush(x.display)
	return nil
}

// typeUnicodeHex types a Unicode character via Ctrl+Shift+U + hex + Enter.
// This works on most Linux desktops (GNOME, KDE, etc.).
func (x *xtestDriver) typeUnicodeHex(display *C.Display, r rune) {
	ctrlKC := C.XKeysymToKeycode(display, C.XK_Control_L)
	shiftKC := C.XKeysymToKeycode(display, C.XK_Shift_L)
	uKC := C.XKeysymToKeycode(display, C.XK_U)
	returnKC := C.XKeysymToKeycode(display, C.XK_Return)

	if ctrlKC == 0 || shiftKC == 0 || uKC == 0 || returnKC == 0 {
		return
	}

	// Ctrl+Shift+U
	C.xFakeKey(display, ctrlKC, 1)
	C.xFakeKey(display, shiftKC, 1)
	C.xFakeKey(display, uKC, 1)
	C.xFakeKey(display, uKC, 0)
	C.xFakeKey(display, shiftKC, 0)
	C.xFakeKey(display, ctrlKC, 0)

	// Type hex digits
	hex := fmt.Sprintf("%x", r)
	for _, ch := range hex {
		var code C.KeyCode
		if ch >= '0' && ch <= '9' {
			code = C.XKeysymToKeycode(display, C.KeySym('0'+C.int(ch-'0')))
		} else {
			code = C.XKeysymToKeycode(display, C.KeySym(C.int(ch)))
		}
		if code != 0 {
			C.xFakeKey(display, code, 1)
			C.xFakeKey(display, code, 0)
		}
	}

	// Enter to confirm
	C.xFakeKey(display, returnKC, 1)
	C.xFakeKey(display, returnKC, 0)
}

func (x *xtestDriver) KeyText(text string) error {
	if text == "" {
		return nil
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	display := x.display

	shiftKC := C.XKeysymToKeycode(display, C.XK_Shift_L)
	altgrKC := C.XKeysymToKeycode(display, C.XK_Alt_R)
	hasAltGr := altgrKC != 0

	for _, r := range text {
		if r == '\n' {
			enterKC := C.XKeysymToKeycode(display, C.XK_Return)
			if enterKC != 0 {
				C.xFakeKey(display, enterKC, 1)
				C.xFakeKey(display, enterKC, 0)
				C.XFlush(display)
			}
			continue
		}

		target := C.KeySym(r)
		if target == 0 {
			continue
		}

		var shift, altgr C.int
		kc := C.findKeycode(display, target, &shift, &altgr)
		if kc == 0 {
			cs := C.CString(string(r))
			target = C.XStringToKeysym(cs)
			C.free(unsafe.Pointer(cs))
			if target == C.NoSymbol {
				x.typeUnicodeHex(display, r)
				C.XFlush(display)
				continue
			}
			kc = C.findKeycode(display, target, &shift, &altgr)
			if kc == 0 {
				x.typeUnicodeHex(display, r)
				C.XFlush(display)
				continue
			}
		}

		// If character needs AltGr but AltGr key doesn't exist on this layout,
		// fall back to Unicode hex input
		if altgr != 0 && !hasAltGr {
			x.typeUnicodeHex(display, r)
			C.XFlush(display)
			continue
		}

		useShift := shift != 0 && shiftKC != 0
		useAltGr := altgr != 0 && hasAltGr

		if useShift {
			C.xFakeKey(display, shiftKC, 1)
		}
		if useAltGr {
			C.xFakeKey(display, altgrKC, 1)
		}

		C.xFakeKey(display, kc, 1)
		C.xFakeKey(display, kc, 0)

		if useAltGr {
			C.xFakeKey(display, altgrKC, 0)
		}
		if useShift {
			C.xFakeKey(display, shiftKC, 0)
		}

		// Flush after each character to ensure proper key state sync
		C.XFlush(display)
	}
	return nil
}

func (x *xtestDriver) keyCommand(key string, down, up bool) error {
	code := x.keyNameToKeycode(key)
	if code == 0 {
		return nil
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if down {
		C.xFakeKey(x.display, code, 1)
	}
	if up {
		C.xFakeKey(x.display, code, 0)
	}
	if down || up {
		C.XFlush(x.display)
	}
	return nil
}

func (x *xtestDriver) Key(key string) error {
	return x.keyCommand(key, true, true)
}

func (x *xtestDriver) KeyDown(key string) error {
	return x.keyCommand(key, true, false)
}

func (x *xtestDriver) KeyUp(key string) error {
	return x.keyCommand(key, false, true)
}

func (x *xtestDriver) Hotkey(mods []string, key string) error {
	codes := x.modCodes(mods)
	mainCode := x.keyNameToKeycode(key)

	x.mu.Lock()
	defer x.mu.Unlock()

	for _, c := range codes {
		C.xFakeKey(x.display, c, 1)
	}
	if mainCode != 0 {
		C.xFakeKey(x.display, mainCode, 1)
		C.xFakeKey(x.display, mainCode, 0)
	}
	for i := len(codes) - 1; i >= 0; i-- {
		C.xFakeKey(x.display, codes[i], 0)
	}
	C.XFlush(x.display)
	return nil
}

func (x *xtestDriver) vkCommand(k KeySpec, down, up bool) error {
	code := x.vkToKeycode(k.VK)
	if code == 0 {
		return nil
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if down {
		C.xFakeKey(x.display, code, 1)
	}
	if up {
		C.xFakeKey(x.display, code, 0)
	}
	if down || up {
		C.XFlush(x.display)
	}
	return nil
}

func (x *xtestDriver) KeyVK(k KeySpec) error {
	return x.vkCommand(k, true, true)
}

func (x *xtestDriver) KeyDownVK(k KeySpec) error {
	return x.vkCommand(k, true, false)
}

func (x *xtestDriver) KeyUpVK(k KeySpec) error {
	return x.vkCommand(k, false, true)
}

func (x *xtestDriver) HotkeyVK(mods []string, k KeySpec) error {
	codes := x.modCodes(mods)
	mainCode := x.vkToKeycode(k.VK)

	x.mu.Lock()
	defer x.mu.Unlock()

	for _, c := range codes {
		C.xFakeKey(x.display, c, 1)
	}
	if mainCode != 0 {
		C.xFakeKey(x.display, mainCode, 1)
		C.xFakeKey(x.display, mainCode, 0)
	}
	for i := len(codes) - 1; i >= 0; i-- {
		C.xFakeKey(x.display, codes[i], 0)
	}
	C.XFlush(x.display)
	return nil
}

func (x *xtestDriver) modCodes(mods []string) []C.KeyCode {
	var codes []C.KeyCode
	for _, m := range mods {
		code := x.modNameToKeycode(m)
		if code != 0 {
			codes = append(codes, code)
		}
	}
	return codes
}

func (x *xtestDriver) modNameToKeycode(m string) C.KeyCode {
	switch strings.ToLower(m) {
	case "ctrl", "control":
		return C.XKeysymToKeycode(x.display, C.XK_Control_L)
	case "alt":
		return C.XKeysymToKeycode(x.display, C.XK_Alt_L)
	case "shift":
		return C.XKeysymToKeycode(x.display, C.XK_Shift_L)
	case "win", "windows", "meta":
		return C.XKeysymToKeycode(x.display, C.XK_Super_L)
	default:
		return 0
	}
}

func (x *xtestDriver) keyNameToKeycode(key string) C.KeyCode {
	switch strings.ToLower(key) {
	case "enter":
		return C.XKeysymToKeycode(x.display, C.XK_Return)
	case "backspace":
		return C.XKeysymToKeycode(x.display, C.XK_BackSpace)
	case "tab":
		return C.XKeysymToKeycode(x.display, C.XK_Tab)
	case "esc", "escape":
		return C.XKeysymToKeycode(x.display, C.XK_Escape)
	case "space":
		return C.XKeysymToKeycode(x.display, C.XK_space)
	case "up":
		return C.XKeysymToKeycode(x.display, C.XK_Up)
	case "down":
		return C.XKeysymToKeycode(x.display, C.XK_Down)
	case "left":
		return C.XKeysymToKeycode(x.display, C.XK_Left)
	case "right":
		return C.XKeysymToKeycode(x.display, C.XK_Right)
	case "delete", "del":
		return C.XKeysymToKeycode(x.display, C.XK_Delete)
	case "win", "windows":
		return C.XKeysymToKeycode(x.display, C.XK_Super_L)
	case "vol_mute":
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioMute)
	case "vol_down":
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioLowerVolume)
	case "vol_up":
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioRaiseVolume)
	case "media_next":
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioNext)
	case "media_prev":
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioPrev)
	case "media_play_pause":
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioPlay)
	}

	if len(key) == 1 {
		c := key[0]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return C.XKeysymToKeycode(x.display, C.KeySym(c))
		}
		if c >= '0' && c <= '9' {
			return C.XKeysymToKeycode(x.display, C.KeySym(c))
		}
	}

	if strings.HasPrefix(key, "f") && len(key) <= 3 {
		n := 0
		for _, ch := range key[1:] {
			if ch < '0' || ch > '9' {
				n = 0
				break
			}
			n = n*10 + int(ch-'0')
		}
		if n >= 1 && n <= 12 {
			return C.XKeysymToKeycode(x.display, C.KeySym(C.XK_F1+(n-1)))
		}
	}

	return 0
}

func (x *xtestDriver) vkToKeycode(vk uint16) C.KeyCode {
	switch vk {
	case 0x08:
		return C.XKeysymToKeycode(x.display, C.XK_BackSpace)
	case 0x09:
		return C.XKeysymToKeycode(x.display, C.XK_Tab)
	case 0x0D:
		return C.XKeysymToKeycode(x.display, C.XK_Return)
	case 0x10:
		return C.XKeysymToKeycode(x.display, C.XK_Shift_L)
	case 0x11:
		return C.XKeysymToKeycode(x.display, C.XK_Control_L)
	case 0x12:
		return C.XKeysymToKeycode(x.display, C.XK_Alt_L)
	case 0x1B:
		return C.XKeysymToKeycode(x.display, C.XK_Escape)
	case 0x20:
		return C.XKeysymToKeycode(x.display, C.XK_space)
	case 0x21:
		return C.XKeysymToKeycode(x.display, C.XK_Page_Up)
	case 0x22:
		return C.XKeysymToKeycode(x.display, C.XK_Page_Down)
	case 0x23:
		return C.XKeysymToKeycode(x.display, C.XK_End)
	case 0x24:
		return C.XKeysymToKeycode(x.display, C.XK_Home)
	case 0x25:
		return C.XKeysymToKeycode(x.display, C.XK_Left)
	case 0x26:
		return C.XKeysymToKeycode(x.display, C.XK_Up)
	case 0x27:
		return C.XKeysymToKeycode(x.display, C.XK_Right)
	case 0x28:
		return C.XKeysymToKeycode(x.display, C.XK_Down)
	case 0x2D:
		return C.XKeysymToKeycode(x.display, C.XK_Insert)
	case 0x2E:
		return C.XKeysymToKeycode(x.display, C.XK_Delete)
	case 0x5B:
		return C.XKeysymToKeycode(x.display, C.XK_Super_L)
	case 0x5C:
		return C.XKeysymToKeycode(x.display, C.XK_Super_R)
	case 0xAD:
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioMute)
	case 0xAE:
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioLowerVolume)
	case 0xAF:
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioRaiseVolume)
	case 0xB0:
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioNext)
	case 0xB1:
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioPrev)
	case 0xB3:
		return C.XKeysymToKeycode(x.display, C.XF86XK_AudioPlay)
	}

	if vk >= 0x70 && vk <= 0x7B {
		return C.XKeysymToKeycode(x.display, C.KeySym(C.XK_F1+(vk-0x70)))
	}
	if vk >= 'A' && vk <= 'Z' {
		return C.XKeysymToKeycode(x.display, C.KeySym(vk))
	}
	if vk >= '0' && vk <= '9' {
		return C.XKeysymToKeycode(x.display, C.KeySym(vk))
	}

	return 0
}

func mouseXButton(button string) int {
	switch strings.ToLower(button) {
	case "right":
		return 3
	case "middle":
		return 2
	default:
		return 1
	}
}
