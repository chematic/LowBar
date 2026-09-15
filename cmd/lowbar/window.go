//go:build windows

package main

import "syscall"

func requestExit(hwnd syscall.Handle) {
	if shuttingDown {
		return
	}
	shuttingDown = true
	postMessageW.Call(uintptr(hwnd), wmClose, 0, 0)
}

func windowProc(hwnd uintptr, message uint32, wParam uintptr, lParam uintptr) uintptr {
	defer recoverPanic("window procedure")
	switch message {
	case wmTray:
		event := uint32(lParam & 0xFFFF)
		if event == wmRButtonUp || event == wmLButtonDbl {
			if globalConfig != nil && !shuttingDown && !menuPosted {
				menuPosted = true
				postMessageW.Call(hwnd, wmShowMenu, 0, 0)
			}
			return 0
		}
		return 0
	case wmShowMenu:
		menuPosted = false
		if globalConfig != nil && !shuttingDown {
			showTrayMenu(syscall.Handle(hwnd), globalConfig)
		}
		return 0
	case wmCommand:
		return 0
	case wmClose:
		destroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		postQuitMessage.Call(0)
		return 0
	}
	if hookBlockMsg != 0 && message == hookBlockMsg {
		if globalConfig != nil && !shuttingDown {
			return 1
		}
		return 0
	}
	if taskbarMsg != 0 && message == taskbarMsg && globalConfig != nil && !shuttingDown {
		reinstallTrayIcon(syscall.Handle(hwnd))
		installExplorerHookForTaskbarCreation()
		applyStyle(globalConfig.style)
		return 0
	}
	result, _, _ := defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}
