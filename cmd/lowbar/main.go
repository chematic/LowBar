//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"unsafe"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--shutdown" {
		requestExistingInstanceExit()
		return
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	openLog()
	logContext()
	defer func() {
		if v := recover(); v != nil {
			logEvent("FATAL", "main", fmt.Sprintf("panic=%v | stack=\n%s", v, debug.Stack()))
		}
		cleanup()
	}()
	prepareProcess()
	processMutex = ensureSingleInstance()
	if processMutex == 0 {
		return
	}
	cfg := loadConfig()
	startup := isStartupEnabled()
	if cfg.startup != startup {
		cfg.startup = startup
		_ = saveConfig(cfg)
	}
	globalConfig = &cfg
	taskbarRaw, _, _ := registerWindowMessageW.Call(uintptr(unsafe.Pointer(utf16ptr(taskbarMessage))))
	taskbarMsg = uint32(taskbarRaw)
	if err := loadTrayIcon(); err != nil {
		showError(appName, err)
		return
	}
	hwnd, err := createHiddenWindow()
	if err != nil {
		showError(appName, err)
		return
	}
	defer destroyWindow.Call(uintptr(hwnd))
	tray, err := installTrayIcon(hwnd)
	if err != nil {
		showError(appName, err)
		return
	}
	trayData = &tray
	applyStyle(cfg.style)
	startTaskbarMonitor(hwnd)
	if err := saveConfig(cfg); err != nil {
		logError("config.save", "initial config save failed", err)
	}
	_ = messageLoop()
}
