//go:build windows

package input

import (
	"errors"
	"strings"
	"syscall"
	"unsafe"
)

var (
	enumWindows              = user32.NewProc("EnumWindows")
	isWindowVisible          = user32.NewProc("IsWindowVisible")
	getWindowTextW           = user32.NewProc("GetWindowTextW")
	getWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	getWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	getWindow                = user32.NewProc("GetWindow")
	getWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")
	isIconic                 = user32.NewProc("IsIconic")
	showWindow               = user32.NewProc("ShowWindow")
	setForegroundWindow      = user32.NewProc("SetForegroundWindow")
	postMessageW             = user32.NewProc("PostMessageW")
	bringWindowToTop         = user32.NewProc("BringWindowToTop")

	kernel32dll                = syscall.NewLazyDLL("kernel32.dll")
	openProcess                = kernel32dll.NewProc("OpenProcess")
	closeHandle                = kernel32dll.NewProc("CloseHandle")
	queryFullProcessImageNameW = kernel32dll.NewProc("QueryFullProcessImageNameW")
)

const (
	GW_OWNER = 4

	// IMPORTANT: must be signed, because it's negative
	GWL_EXSTYLE int32 = -20

	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_APPWINDOW  = 0x00040000

	SW_SHOW          = 5
	SW_MINIMIZE      = 6
	SW_SHOWMAXIMIZED = 3
	SW_RESTORE       = 9

	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

	WM_CLOSE = 0x0010
)

// Convert signed int32 index to uintptr safely (works on Windows x64 where int is 64-bit)
func ptrIndex(v int32) uintptr { return uintptr(int(v)) }

func windowText(hwnd uintptr) string {
	l, _, _ := getWindowTextLengthW.Call(hwnd)
	if l == 0 {
		return ""
	}
	buf := make([]uint16, l+1)
	getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(l+1))
	return syscall.UTF16ToString(buf)
}

func exePathFromPID(pid uint32) string {
	if pid == 0 {
		return ""
	}
	h, _, _ := openProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer closeHandle.Call(h)

	buf := make([]uint16, 4096)
	sz := uint32(len(buf))
	r, _, _ := queryFullProcessImageNameW.Call(
		h,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&sz)),
	)
	if r == 0 || sz == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:sz])
}

func shouldIncludeWindow(hwnd uintptr) bool {
	v, _, _ := isWindowVisible.Call(hwnd)
	if v == 0 {
		return false
	}

	title := strings.TrimSpace(windowText(hwnd))
	if title == "" {
		return false
	}

	// Typical taskbar heuristic: no owner window
	owner, _, _ := getWindow.Call(hwnd, GW_OWNER)
	if owner != 0 {
		return false
	}

	// Get extended style safely (GWL_EXSTYLE is negative)
	ex, _, _ := getWindowLongPtrW.Call(hwnd, ptrIndex(GWL_EXSTYLE))
	exStyle := uint32(ex)

	// Tool windows usually should not show, unless explicitly appwindow
	if (exStyle&WS_EX_TOOLWINDOW) != 0 && (exStyle&WS_EX_APPWINDOW) == 0 {
		return false
	}

	return true
}

func (w *WindowsInput) ListApps() ([]AppInfo, error) {
	apps := make([]AppInfo, 0, 32)

	cb := syscall.NewCallback(func(hwnd, lparam uintptr) uintptr {
		if !shouldIncludeWindow(hwnd) {
			return 1
		}

		var pid uint32
		getWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		minimized, _, _ := isIconic.Call(hwnd)

		apps = append(apps, AppInfo{
			Hwnd:      hwnd,
			PID:       pid,
			Title:     strings.TrimSpace(windowText(hwnd)),
			Exe:       exePathFromPID(pid),
			Minimized: minimized != 0,
		})
		return 1
	})

	r, _, err := enumWindows.Call(cb, 0)
	if r == 0 {
		if err != nil && err != syscall.Errno(0) {
			return nil, err
		}
		return nil, errors.New("EnumWindows failed")
	}

	return apps, nil
}

func (w *WindowsInput) AppAction(hwnd uintptr, action string) error {
	if hwnd == 0 {
		return errors.New("invalid hwnd")
	}
	switch strings.ToLower(action) {
	case "minimize":
		showWindow.Call(hwnd, SW_MINIMIZE)
		return nil
	case "restore":
		showWindow.Call(hwnd, SW_RESTORE)
		return nil
	case "activate":
		// If minimized, restore first; then bring to top + focus.
		min, _, _ := isIconic.Call(hwnd)
		if min != 0 {
			showWindow.Call(hwnd, SW_RESTORE)
		} else {
			showWindow.Call(hwnd, SW_SHOW)
		}
		bringWindowToTop.Call(hwnd)
		setForegroundWindow.Call(hwnd)
		return nil
	case "maximize":
		showWindow.Call(hwnd, SW_SHOWMAXIMIZED)
		bringWindowToTop.Call(hwnd)
		setForegroundWindow.Call(hwnd)
		return nil
	case "close":
		postMessageW.Call(hwnd, WM_CLOSE, 0, 0)
		return nil
	default:
		return errors.New("unknown action")
	}
}
