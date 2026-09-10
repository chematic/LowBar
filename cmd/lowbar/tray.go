//go:build windows

package main

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

func installTrayIcon(hwnd syscall.Handle) (notifyIconData, error) {
	var data notifyIconData
	data.cbSize = uint32(unsafe.Sizeof(data))
	data.hWnd = hwnd
	data.uID = 1
	data.uFlags = nifMessage | nifIcon | nifTip
	data.uCallbackMessage = wmTray
	data.hIcon = trayIcon
	utf16from(appName, data.szTip[:])
	ret, _, err := shellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&data)))
	if ret == 0 {
		logWin32Error("tray.install", "Shell_NotifyIconW(NIM_ADD)", err)
		if err != nil {
			return data, err
		}
		return data, errors.New("Shell_NotifyIconW failed")
	}
	data.uTimeoutOrVersion = notifyIconVersion4
	if ret, _, err = shellNotifyIconW.Call(nimSetVersion, uintptr(unsafe.Pointer(&data))); ret == 0 {
		logWin32Error("tray.install", "Shell_NotifyIconW(NIM_SETVERSION)", err)
	}
	return data, nil
}

func removeTrayIcon(data *notifyIconData) {
	if data == nil || data.hWnd == 0 {
		return
	}
	shellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(data)))
}

func reinstallTrayIcon(hwnd syscall.Handle) {
	if trayData != nil {
		removeTrayIcon(trayData)
	}
	data, err := installTrayIcon(hwnd)
	if err != nil {
		logError("tray.reinstall", "unable to reinstall tray icon", err)
		trayData = nil
		return
	}
	trayData = &data
	logEvent("INFO", "tray.reinstall", "tray icon reinstalled after Explorer refresh")
}

func appendMenu(menu, flags, id uintptr, text string) bool {
	ret, _, err := appendMenuW.Call(menu, flags, id, uintptr(unsafe.Pointer(utf16ptr(text))))
	if ret == 0 {
		logWin32Error("tray.menu", fmt.Sprintf("AppendMenuW menu=%d id=%d text=%q", menu, id, text), err)
		return false
	}
	return true
}

func menuCheckFlag(active bool) uintptr {
	if active {
		return mfChecked
	}
	return 0
}
