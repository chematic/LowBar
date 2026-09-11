//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

func ensureSingleInstance() syscall.Handle {
	mutex, _, _ := createMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16ptr(mutexName))))
	if mutex == 0 {
		return 0
	}
	last, _, _ := getLastError.Call()
	if last == errorAlreadyExists {
		closeHandle.Call(mutex)
		return 0
	}
	return syscall.Handle(mutex)
}

func requestExistingInstanceExit() {
	className := utf16ptr(windowClass)
	hwnd, _, _ := findWindowW.Call(uintptr(unsafe.Pointer(className)), 0)
	if hwnd == 0 {
		return
	}
	postMessageW.Call(hwnd, wmClose, 0, 0)
}
