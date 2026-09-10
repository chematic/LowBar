//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func taskbarWindows() []syscall.Handle {
	var result []syscall.Handle
	primaryName := utf16ptr("Shell_TrayWnd")
	secondaryName := utf16ptr("Shell_SecondaryTrayWnd")
	primary, _, _ := findWindowW.Call(uintptr(unsafe.Pointer(primaryName)), 0)
	if primary != 0 {
		result = append(result, syscall.Handle(primary))
	}
	var after uintptr
	for {
		hwnd, _, _ := findWindowExW.Call(0, after, uintptr(unsafe.Pointer(secondaryName)), 0)
		if hwnd == 0 {
			break
		}
		result = append(result, syscall.Handle(hwnd))
		after = hwnd
	}
	return result
}

func accentForStyle(style int) accentPolicy {
	switch style {
	case styleNormal:
		return accentPolicy{state: 0, flags: 0, gradient: 0, animation: 0}
	case styleOpaque:
		return accentPolicy{state: 1, flags: 2, gradient: 0xFF111111}
	case styleClear:
		return accentPolicy{state: 2, flags: 2, gradient: 0x00000000}
	case styleBlur:
		return accentPolicy{state: 3, flags: 2, gradient: 0x55000000}
	case styleAcrylic:
		return accentPolicy{state: 4, flags: 2, gradient: 0xAA000000}
	default:
		return accentPolicy{state: 0}
	}
}

func setAccent(hwnd syscall.Handle, style int) bool {
	if hwnd == 0 {
		return false
	}
	policy := accentForStyle(style)
	data := compositionAttributeData{attribute: 19, data: uintptr(unsafe.Pointer(&policy)), dataSize: unsafe.Sizeof(policy)}
	ret, _, err := setWindowComposition.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&data)))
	if ret == 0 {
		logWin32Error("taskbar.setAccent", fmt.Sprintf("SetWindowCompositionAttribute hwnd=%d style=%d", hwnd, style), err)
		return false
	}
	return true
}

func applyStyle(style int) {
	defer recoverPanic("applyStyle")
	if !validStyle(style) {
		logEvent("ERROR", "taskbar.applyStyle", fmt.Sprintf("invalid style=%d; falling back to Normal", style))
		style = styleNormal
	}
	windows := taskbarWindows()
	if len(windows) == 0 {
		logEvent("ERROR", "taskbar.applyStyle", "no taskbar windows found")
		return
	}
	success := 0
	for _, hwnd := range windows {
		if setAccent(hwnd, style) {
			success++
		}
	}
	if success == len(windows) {
		processStyle = style
		styleApplied = style != styleNormal
		logEvent("INFO", "taskbar.applyStyle", fmt.Sprintf("style=%d applied to %d taskbar window(s)", style, success))
	} else {
		logEvent("ERROR", "taskbar.applyStyle", fmt.Sprintf("style=%d partially applied success=%d total=%d", style, success, len(windows)))
	}
}

func restoreTaskbar() {
	defer recoverPanic("restoreTaskbar")
	if !styleApplied {
		return
	}
	windows := taskbarWindows()
	for _, hwnd := range windows {
		_ = setAccent(hwnd, styleNormal)
	}
	styleApplied = false
	processStyle = styleNormal
	logEvent("INFO", "taskbar.restore", "taskbar restored to Normal/Windows default")
}
