package input

// KeySpec is a stable representation of a key that can be stored on the phone
// and later sent back to the daemon to be executed.
//
// VK is the Windows Virtual-Key code.
// Scan is the hardware scan code (optional; can be 0).
// Ext marks extended keys (arrows, insert/delete, etc.).
type KeySpec struct {
	VK   uint16 `json:"vk"`
	Scan uint16 `json:"scan,omitempty"`
	Ext  bool   `json:"ext,omitempty"`
}

// CaptureResult is what the daemon returns after the user presses a key while
// capture mode is active.
type CaptureResult struct {
	Key  KeySpec  `json:"key"`
	Mods []string `json:"mods,omitempty"` // ctrl|alt|shift|win
}

// AppInfo represents a top-level application window that typically appears in
// the taskbar / Alt-Tab list.
type AppInfo struct {
	Hwnd      uintptr `json:"hwnd"`
	PID       uint32  `json:"pid"`
	Title     string  `json:"title"`
	Exe       string  `json:"exe,omitempty"`
	Minimized bool    `json:"minimized"`
}

type InputDriver interface {
	MoveMouse(dx, dy int32) error

	MouseClick(button string) error
	MouseDown(button string) error
	MouseUp(button string) error
	MouseScroll(dy int32) error

	KeyText(text string) error
	Key(key string) error
	KeyDown(key string) error
	KeyUp(key string) error
	Hotkey(mods []string, key string) error

	// Stable key execution (phone stores VK/Scan instead of daemon-defined names)
	KeyVK(k KeySpec) error
	KeyDownVK(k KeySpec) error
	KeyUpVK(k KeySpec) error
	HotkeyVK(mods []string, k KeySpec) error

	// One-shot key capture (no keylogger). Blocks until a key is pressed or timeoutMs elapses.
	CaptureNextKey(timeoutMs int) (CaptureResult, error)

	// Taskbar apps
	ListApps() ([]AppInfo, error)
	AppAction(hwnd uintptr, action string) error // minimize|restore|activate|maximize|close
}
