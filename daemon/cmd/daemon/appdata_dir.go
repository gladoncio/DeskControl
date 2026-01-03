package main

import (
	"os"
	"path/filepath"
)

// appDataDir returns a directory for app data.
// Prefer OS config dir, fallback to executable directory.
func appDataDir() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		p := filepath.Join(dir, "DeskControl")
		_ = os.MkdirAll(p, 0o755)
		return p
	}

	exe, err := os.Executable()
	if err == nil && exe != "" {
		p := filepath.Join(filepath.Dir(exe), "data")
		_ = os.MkdirAll(p, 0o755)
		return p
	}

	p := "data"
	_ = os.MkdirAll(p, 0o755)
	return p
}
