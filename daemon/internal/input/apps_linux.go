//go:build linux

package input

import (
	"errors"
	"log"
)

func listApps() ([]AppInfo, error) {
	apps, err := x11ListApps()
	if err == nil {
		return apps, nil
	}
	log.Printf("[apps] X11 listing failed: %v, trying D-Bus", err)
	return dbusListApps()
}

func appAction(hwnd uintptr, action string) error {
	if hwnd == 0 {
		return errors.New("invalid window id")
	}
	return x11AppAction(hwnd, action)
}
