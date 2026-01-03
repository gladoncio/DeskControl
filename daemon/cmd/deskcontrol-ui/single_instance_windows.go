//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW     = kernel32.NewProc("CreateMutexW")
	procGetLastError     = kernel32.NewProc("GetLastError")
	ERROR_ALREADY_EXISTS = uintptr(183)
)

func ensureSingleInstance(name string) bool {
	n, _ := syscall.UTF16PtrFromString(name)
	h, _, _ := procCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(n)))
	if h == 0 {
		// si falla, igual dejamos correr
		return true
	}
	e, _, _ := procGetLastError.Call()
	if e == ERROR_ALREADY_EXISTS {
		return false
	}
	return true
}
