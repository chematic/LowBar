//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"
)

func appDataDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.Getenv("APPDATA")
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, configDirName)
}

func configPath() string { return filepath.Join(appDataDir(), configFileName) }
func logDir() string     { return filepath.Join(os.TempDir(), configDirName) }
func logPath() string    { return filepath.Join(logDir(), logFileName) }

func openLog() {
	if err := os.MkdirAll(logDir(), 0700); err != nil {
		return
	}
	f, err := os.OpenFile(logPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		logHandle = f
	}
}

func logEvent(level, area, message string) {
	if logHandle == nil {
		return
	}
	line := fmt.Sprintf("%s [%s] [%s] %s\r\n", time.Now().Format("2006-01-02 15:04:05.000 -0700"), level, area, message)
	_, _ = logHandle.WriteString(line)
	_ = logHandle.Sync()
}

func logLine(s string) { logEvent("INFO", "general", s) }

func logError(area, message string, err error) {
	if err == nil {
		logEvent("ERROR", area, message)
		return
	}
	logEvent("ERROR", area, message+" | error="+err.Error())
}

func logWin32Error(area, operation string, procErr error) {
	last, _, _ := getLastError.Call()
	if last == 0 {
		logEvent("ERROR", area, fmt.Sprintf("operation=%s | procError=%v | win32LastError=0", operation, procErr))
		return
	}
	logEvent("ERROR", area, fmt.Sprintf("operation=%s | procError=%v | win32LastError=%d (%v)", operation, procErr, last, syscall.Errno(last)))
}

func executablePath() string {
	e, err := os.Executable()
	if err != nil {
		return "<unavailable: " + err.Error() + ">"
	}
	return e
}

func logContext() {
	logEvent("INFO", "startup", fmt.Sprintf("version=dev | go=%s | arch=%s | os=%s | pid=%d | exe=%s | temp=%s | config=%s | log=%s", runtime.Version(), runtime.GOARCH, runtime.GOOS, os.Getpid(), executablePath(), os.TempDir(), configPath(), logPath()))
}

func closeLog() {
	if logHandle != nil {
		_ = logHandle.Close()
		logHandle = nil
	}
}

func recoverPanic(area string) {
	if v := recover(); v != nil {
		logEvent("FATAL", area, fmt.Sprintf("panic=%v | stack=\n%s", v, debug.Stack()))
	}
}
