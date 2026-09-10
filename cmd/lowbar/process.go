//go:build windows

package main

import (
	"errors"
	"syscall"
	"unsafe"
)

func showError(title string, err error) {
	if err == nil {
		err = errors.New("unknown error")
	}
	logError("ui.error", title, err)
	messageBoxW.Call(0, uintptr(unsafe.Pointer(utf16ptr(err.Error()))), uintptr(unsafe.Pointer(utf16ptr(title))), messageBoxError)
}

func showInfo(owner syscall.Handle, title, text string) {
	messageBoxW.Call(uintptr(owner), uintptr(unsafe.Pointer(utf16ptr(text))), uintptr(unsafe.Pointer(utf16ptr(title))), messageBoxInfo)
}

func getModuleHandle() syscall.Handle {
	v, _, _ := getModuleHandleW.Call(0)
	return syscall.Handle(v)
}

func prepareProcess() {
	if ret, _, err := setCurrentProcessAppID.Call(uintptr(unsafe.Pointer(utf16ptr(appUserModelID)))); ret == 0 {
		logWin32Error("process", "SetCurrentProcessExplicitAppUserModelID", err)
	}
	if ret, _, err := setProcessDpiAwarenessCtx.Call(uintptr(^uintptr(0))); ret == 0 {
		logWin32Error("process", "SetProcessDpiAwarenessContext", err)
	}
	if ret, _, err := setProcessDPIAware.Call(); ret == 0 {
		logWin32Error("process", "SetProcessDPIAware", err)
	}
	if ret, _, err := setThreadDpiAwareness.Call(uintptr(^uintptr(0))); ret == 0 {
		logWin32Error("process", "SetThreadDpiAwarenessContext", err)
	}
}

func createHiddenWindow() (syscall.Handle, error) {
	instance := getModuleHandle()
	className := utf16ptr(windowClass)
	cursor, _, _ := loadCursorW.Call(0, cursorArrow)
	wc := wndClassEx{cbSize: uint32(unsafe.Sizeof(wndClassEx{})), style: csHRedraw | csVRedraw, lpfnWndProc: syscall.NewCallback(windowProc), hInstance: instance, hIcon: trayIcon, hCursor: syscall.Handle(cursor), lpszClassName: className, hIconSm: trayIcon}
	ret, _, callErr := registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 && callErr != nil && !errors.Is(callErr, syscall.Errno(errorClassAlreadyReg)) {
		return 0, callErr
	}
	hwnd, _, err := createWindowExW.Call(wsExToolWindow|wsExNoActivate, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(utf16ptr(appName))), wsPopup, 0, 0, 0, 0, 0, 0, uintptr(instance), 0)
	if hwnd == 0 {
		if err == nil {
			err = syscall.EINVAL
		}
		return 0, err
	}
	return syscall.Handle(hwnd), nil
}
