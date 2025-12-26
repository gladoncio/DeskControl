//go:build windows

package input

import (
	"errors"
	"log"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	WH_KEYBOARD_LL = 13

	WM_KEYDOWN    = 0x0100
	WM_SYSKEYDOWN = 0x0104
)

type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	getModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	getCurrentThreadId = kernel32.NewProc("GetCurrentThreadId") // ✅ FIX: kernel32, no user32

	setWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	unhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	callNextHookEx      = user32.NewProc("CallNextHookEx")
	getMessageW         = user32.NewProc("GetMessageW")
	translateMessage    = user32.NewProc("TranslateMessage")
	dispatchMessageW    = user32.NewProc("DispatchMessageW")
	postThreadMessageW  = user32.NewProc("PostThreadMessageW")
	getAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
)

const (
	LLKHF_EXTENDED = 0x01
)

var (
	capMu       sync.Mutex
	capActive   bool
	capResultCh chan CaptureResult
	capThreadID uint32
)

func asyncDown(vk uint16) bool {
	r, _, _ := getAsyncKeyState.Call(uintptr(vk))
	return (uint16(r) & 0x8000) != 0
}

func currentMods() []string {
	mods := make([]string, 0, 4)
	if asyncDown(0x11) { // VK_CONTROL
		mods = append(mods, "ctrl")
	}
	if asyncDown(0x12) { // VK_MENU
		mods = append(mods, "alt")
	}
	if asyncDown(0x10) { // VK_SHIFT
		mods = append(mods, "shift")
	}
	if asyncDown(0x5B) || asyncDown(0x5C) { // LWIN/RWIN
		mods = append(mods, "win")
	}
	return mods
}

func keyboardHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 {
		msg := uint32(wParam)
		if msg == WM_KEYDOWN || msg == WM_SYSKEYDOWN {
			k := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
			res := CaptureResult{
				Key: KeySpec{
					VK:   uint16(k.VkCode),
					Scan: uint16(k.ScanCode),
					Ext:  (k.Flags & LLKHF_EXTENDED) != 0,
				},
				Mods: currentMods(),
			}

			capMu.Lock()
			ch := capResultCh
			active := capActive
			capMu.Unlock()

			if active && ch != nil {
				select {
				case ch <- res:
				default:
				}
			}
		}
	}
	r, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return r
}

func installKeyboardHook() (uintptr, error) {
	cb := syscall.NewCallback(keyboardHookProc)

	// hMod = GetModuleHandleW(NULL)
	hMod, _, _ := getModuleHandleW.Call(0)
	log.Printf("[capture] GetModuleHandleW(NULL) -> hMod=%d", hMod)

	h, _, err := setWindowsHookExW.Call(
		uintptr(WH_KEYBOARD_LL),
		cb,
		hMod,
		0,
	)
	log.Printf("[capture] SetWindowsHookExW -> h=%d err=%v", h, err)

	if h == 0 {
		if err != nil && err != syscall.Errno(0) {
			return 0, err
		}
		return 0, errors.New("SetWindowsHookExW failed (h=0)")
	}
	return h, nil
}

func uninstallKeyboardHook(h uintptr) {
	if h == 0 {
		return
	}
	r, _, err := unhookWindowsHookEx.Call(h)
	log.Printf("[capture] UnhookWindowsHookEx(h=%d) -> r=%d err=%v", h, r, err)
}

func (w *WindowsInput) CaptureNextKey(timeoutMs int) (CaptureResult, error) {
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}

	capMu.Lock()
	if capActive {
		capMu.Unlock()
		return CaptureResult{}, errors.New("capture already active")
	}
	capActive = true
	capResultCh = make(chan CaptureResult, 1)
	capMu.Unlock()

	log.Printf("[capture] CaptureNextKey begin timeoutMs=%d", timeoutMs)

	defer func() {
		capMu.Lock()
		capActive = false
		capResultCh = nil
		capMu.Unlock()
		log.Printf("[capture] CaptureNextKey end")
	}()

	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[capture] PANIC in capture thread: %v\n%s", rec, string(debug.Stack()))
				select {
				case errCh <- errors.New("panic in capture thread (see daemon logs)"):
				default:
				}
			}
		}()

		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		tid, _, _ := getCurrentThreadId.Call() // ✅ FIXED
		capMu.Lock()
		capThreadID = uint32(tid)
		capMu.Unlock()

		log.Printf("[capture] thread started tid=%d", uint32(tid))

		h, err := installKeyboardHook()
		if err != nil {
			log.Printf("[capture] install hook error: %v", err)
			errCh <- err
			return
		}
		defer uninstallKeyboardHook(h)

		var m MSG
		for {
			r, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			rr := int32(r)
			if rr == -1 {
				log.Printf("[capture] GetMessageW -> -1 (error)")
				break
			}
			if rr == 0 {
				log.Printf("[capture] GetMessageW -> 0 (WM_QUIT)")
				break
			}
			_, _, _ = translateMessage.Call(uintptr(unsafe.Pointer(&m)))
			_, _, _ = dispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}

		log.Printf("[capture] thread exiting tid=%d", uint32(tid))
	}()

	timeout := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timeout.Stop()

	select {
	case err := <-errCh:
		log.Printf("[capture] returning error: %v", err)
		return CaptureResult{}, err

	case res := <-capResultCh:
		log.Printf("[capture] got key vk=%d scan=%d ext=%v mods=%v", res.Key.VK, res.Key.Scan, res.Key.Ext, res.Mods)

		capMu.Lock()
		tid := capThreadID
		capMu.Unlock()

		if tid != 0 {
			r, _, err := postThreadMessageW.Call(uintptr(tid), 0x0012 /*WM_QUIT*/, 0, 0)
			log.Printf("[capture] PostThreadMessageW(tid=%d, WM_QUIT) -> r=%d err=%v", tid, r, err)
		}

		return res, nil

	case <-timeout.C:
		log.Printf("[capture] timeout after %dms", timeoutMs)

		capMu.Lock()
		tid := capThreadID
		capMu.Unlock()

		if tid != 0 {
			r, _, err := postThreadMessageW.Call(uintptr(tid), 0x0012 /*WM_QUIT*/, 0, 0)
			log.Printf("[capture] PostThreadMessageW(tid=%d, WM_QUIT) -> r=%d err=%v", tid, r, err)
		}

		return CaptureResult{}, errors.New("capture timeout")
	}
}
