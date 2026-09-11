//go:build windows

package main

import "syscall"

func startTaskbarMonitor(hwnd syscall.Handle) {
	if hwnd == 0 || taskbarTimer {
		return
	}
	ret, _, err := setTimer.Call(uintptr(hwnd), taskbarTimerID, taskbarTimerInterval, 0)
	if ret == 0 {
		logWin32Error("taskbar.monitor", "SetTimer", err)
		return
	}
	taskbarTimer = true
	logEvent("INFO", "taskbar.monitor", "taskbar style monitor started")
}

func stopTaskbarMonitor(hwnd syscall.Handle) {
	if hwnd == 0 || !taskbarTimer {
		return
	}
	ret, _, err := killTimer.Call(uintptr(hwnd), taskbarTimerID)
	if ret == 0 {
		logWin32Error("taskbar.monitor", "KillTimer", err)
	}
	taskbarTimer = false
	logEvent("INFO", "taskbar.monitor", "taskbar style monitor stopped")
}
