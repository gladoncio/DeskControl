//go:build unix

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func ensureSingleInstance(name string) bool {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return true
	}
	lockDir := filepath.Join(cacheDir, name)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return true
	}
	lockPath := filepath.Join(lockDir, "instance.lock")

	ln, err := net.Listen("unix", lockPath)
	if err != nil {
		fmt.Printf("Another instance is already running (lock: %s)\n", lockPath)
		return false
	}
	ln.Close()

	os.Remove(lockPath)
	return true
}
