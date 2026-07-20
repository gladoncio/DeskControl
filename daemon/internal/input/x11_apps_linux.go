//go:build linux

package input

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/Xutil.h>

static int sendCloseEvent(Display *d, Window w) {
    Atom netClose = XInternAtom(d, "_NET_CLOSE_WINDOW", 0);
    if (netClose == 0) return -1;

    XEvent ev;
    ev.xclient.type = ClientMessage;
    ev.xclient.window = w;
    ev.xclient.message_type = netClose;
    ev.xclient.format = 32;
    ev.xclient.data.l[0] = CurrentTime;
    ev.xclient.data.l[1] = 0;
    ev.xclient.data.l[2] = 0;
    ev.xclient.data.l[3] = 0;
    ev.xclient.data.l[4] = 0;

    return XSendEvent(d, XDefaultRootWindow(d), 0,
        SubstructureRedirectMask | SubstructureNotifyMask, &ev);
}

static int maximizeWindow(Display *d, Window w) {
    Atom netWmState = XInternAtom(d, "_NET_WM_STATE", 0);
    Atom maxVert = XInternAtom(d, "_NET_WM_STATE_MAXIMIZED_VERT", 0);
    Atom maxHorz = XInternAtom(d, "_NET_WM_STATE_MAXIMIZED_HORZ", 0);
    if (netWmState == 0 || maxVert == 0 || maxHorz == 0) return -1;

    Atom atoms[2] = {maxVert, maxHorz};
    XChangeProperty(d, w, netWmState, XA_ATOM, 32,
        PropModeReplace, (unsigned char*)atoms, 2);
    return 0;
}
*/
import "C"

import (
	"errors"
	"strings"
	"unsafe"
)

func x11ListApps() ([]AppInfo, error) {
	d := C.XOpenDisplay(nil)
	if d == nil {
		return nil, errors.New("cannot open X display")
	}
	defer C.XCloseDisplay(d)

	root := C.XDefaultRootWindow(d)

	atomNetClientList := C.XInternAtom(d, C.CString("_NET_CLIENT_LIST"), 0)
	if atomNetClientList == 0 {
		return nil, errors.New("_NET_CLIENT_LIST atom not available")
	}

	var actualType C.Atom
	var actualFormat C.int
	var nItems C.ulong
	var bytesAfter C.ulong
	var prop *C.uchar

	status := C.XGetWindowProperty(d, root, atomNetClientList, 0, (1<<31)-1,
		0, C.AnyPropertyType, &actualType, &actualFormat,
		&nItems, &bytesAfter, &prop)
	if status != 0 || prop == nil {
		return nil, errors.New("XGetWindowProperty failed")
	}
	defer C.XFree(unsafe.Pointer(prop))

	winPtr := (*[1 << 30]C.Window)(unsafe.Pointer(prop))
	windows := winPtr[:nItems:nItems]

	var apps []AppInfo
	for _, w := range windows {
		if w == 0 {
			continue
		}
		info := getX11WindowInfo(d, w)
		if info.Title == "" {
			continue
		}
		apps = append(apps, info)
	}

	return apps, nil
}

func getX11WindowInfo(d *C.Display, w C.Window) AppInfo {
	info := AppInfo{Hwnd: uintptr(w)}

	title := getX11TextProperty(d, w, "WM_NAME", "UTF8_STRING")
	if title == "" {
		title = getX11TextProperty(d, w, "WM_NAME", "STRING")
	}
	info.Title = strings.TrimSpace(title)

	return info
}

func getX11TextProperty(d *C.Display, w C.Window, propName string, reqType string) string {
	propAtom := C.XInternAtom(d, C.CString(propName), 1)
	if propAtom == 0 {
		return ""
	}

	var reqAtom C.Atom
	if reqType == "UTF8_STRING" {
		reqAtom = C.XInternAtom(d, C.CString("UTF8_STRING"), 1)
	} else if reqType == "STRING" {
		reqAtom = C.XInternAtom(d, C.CString("STRING"), 1)
	} else {
		reqAtom = C.AnyPropertyType
	}

	var actualType C.Atom
	var actualFormat C.int
	var nItems C.ulong
	var bytesAfter C.ulong
	var prop *C.uchar

	status := C.XGetWindowProperty(d, w, propAtom, 0, 1024, 0,
		reqAtom, &actualType, &actualFormat, &nItems, &bytesAfter, &prop)
	if status != 0 || prop == nil || nItems == 0 {
		return ""
	}
	defer C.XFree(unsafe.Pointer(prop))

	if actualFormat == 8 {
		return C.GoString((*C.char)(unsafe.Pointer(prop)))
	}

	return ""
}

func x11AppAction(hwnd uintptr, action string) error {
	d := C.XOpenDisplay(nil)
	if d == nil {
		return errors.New("cannot open X display")
	}
	defer C.XCloseDisplay(d)

	w := C.Window(hwnd)

	switch strings.ToLower(action) {
	case "minimize":
		C.XIconifyWindow(d, w, C.XDefaultScreen(d))
	case "restore", "activate":
		C.XMapWindow(d, w)
		C.XRaiseWindow(d, w)
		C.XSetInputFocus(d, w, C.RevertToPointerRoot, C.CurrentTime)
	case "maximize":
		C.maximizeWindow(d, w)
		C.XMapWindow(d, w)
	case "close":
		C.sendCloseEvent(d, w)
	default:
		return errors.New("unknown action: " + action)
	}

	C.XFlush(d)
	return nil
}
