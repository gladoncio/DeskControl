//go:build linux

package input

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <X11/keysym.h>
#include <X11/keysymdef.h>

// Helper to get key info from XEvent
static int waitForKeyPress(Display *d, KeySym *ks, unsigned int *state) {
    XEvent ev;
    while (1) {
        XNextEvent(d, &ev);
        if (ev.type == KeyPress) {
            XKeyEvent *ke = (XKeyEvent *)&ev;
            *ks = XKeycodeToKeysym(d, ke->keycode, 0);
            *state = ke->state;
            return (int)ke->keycode;
        }
    }
}

#ifndef XF86XK_AudioMute
#define XF86XK_AudioMute 0x1008FF13
#define XF86XK_AudioLowerVolume 0x1008FF11
#define XF86XK_AudioRaiseVolume 0x1008FF12
#endif
*/
import "C"

import (
	"log"
	"sync"
	"time"
)

type x11Capture struct {
	display *C.Display
	mu      sync.Mutex
}

func (x *x11Capture) init() error {
	d := C.XOpenDisplay(nil)
	if d == nil {
		return errNoDisplay
	}
	x.display = d
	return nil
}

func (x *x11Capture) close() {
	if x.display != nil {
		C.XCloseDisplay(x.display)
	}
}

func (x *x11Capture) run(ch chan<- CaptureResult, stop <-chan struct{}) bool {
	log.Printf("[capture] X11 capture started (XGrabKeyboard)")
	root := C.XDefaultRootWindow(x.display)

	ret := C.XGrabKeyboard(x.display, root, 0, C.GrabModeAsync, C.GrabModeAsync, C.CurrentTime)
	if ret != C.GrabSuccess {
		log.Printf("[capture] XGrabKeyboard failed: %d", ret)
		return false
	}
	defer C.XUngrabKeyboard(x.display, C.CurrentTime)

	// Small delay to let the grab settle
	time.Sleep(50 * time.Millisecond)

	for {
		select {
		case <-stop:
			return true
		default:
		}

		if C.XPending(x.display) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		var ks C.KeySym
		var state C.uint
		kc := C.waitForKeyPress(x.display, &ks, &state)
		if kc <= 0 {
			continue
		}

		res := CaptureResult{
			Key: KeySpec{
				VK:   keysymToVK(ks),
				Scan: uint16(kc),
			},
			Mods: x11Mods(state),
		}

		select {
		case ch <- res:
			return true
		default:
		}
	}
}

var errNoDisplay = errString("no X11 display available")
var errGrabFailed = errString("XGrabKeyboard failed")

type errString string

func (e errString) Error() string { return string(e) }

func keysymToVK(ks C.KeySym) uint16 {
	switch ks {
	case C.XK_BackSpace:
		return 0x08
	case C.XK_Tab:
		return 0x09
	case C.XK_Return:
		return 0x0D
	case C.XK_Escape:
		return 0x1B
	case C.XK_space:
		return 0x20
	case C.XK_Up:
		return 0x26
	case C.XK_Down:
		return 0x28
	case C.XK_Left:
		return 0x25
	case C.XK_Right:
		return 0x27
	case C.XK_Delete:
		return 0x2E
	case C.XK_Home:
		return 0x24
	case C.XK_End:
		return 0x23
	case C.XK_Page_Up:
		return 0x21
	case C.XK_Page_Down:
		return 0x22
	case C.XK_Insert:
		return 0x2D
	case C.XK_Super_L:
		return 0x5B
	case C.XF86XK_AudioMute:
		return 0xAD
	case C.XF86XK_AudioLowerVolume:
		return 0xAE
	case C.XF86XK_AudioRaiseVolume:
		return 0xAF
	}
	if ks >= C.XK_A && ks <= C.XK_Z {
		return uint16(ks)
	}
	if ks >= C.XK_0 && ks <= C.XK_9 {
		return uint16(ks)
	}
	if ks >= C.XK_F1 && ks <= C.XK_F12 {
		return 0x70 + uint16(ks-C.XK_F1)
	}
	return 0
}

func x11Mods(state C.uint) []string {
	var mods []string
	if state&C.ShiftMask != 0 {
		mods = append(mods, "shift")
	}
	if state&C.ControlMask != 0 {
		mods = append(mods, "ctrl")
	}
	if state&C.Mod1Mask != 0 {
		mods = append(mods, "alt")
	}
	if state&C.Mod4Mask != 0 {
		mods = append(mods, "win")
	}
	return mods
}
