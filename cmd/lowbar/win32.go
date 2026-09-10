//go:build windows

package main

import "syscall"

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdiPlus  = syscall.NewLazyDLL("gdiplus.dll")

	registerClassExW          = user32.NewProc("RegisterClassExW")
	createWindowExW           = user32.NewProc("CreateWindowExW")
	defWindowProcW            = user32.NewProc("DefWindowProcW")
	destroyWindow             = user32.NewProc("DestroyWindow")
	getMessageW               = user32.NewProc("GetMessageW")
	translateMessage          = user32.NewProc("TranslateMessage")
	dispatchMessageW          = user32.NewProc("DispatchMessageW")
	postQuitMessage           = user32.NewProc("PostQuitMessage")
	loadIconW                 = user32.NewProc("LoadIconW")
	loadCursorW               = user32.NewProc("LoadCursorW")
	setProcessDPIAware        = user32.NewProc("SetProcessDPIAware")
	setProcessDpiAwarenessCtx = user32.NewProc("SetProcessDpiAwarenessContext")
	setThreadDpiAwareness     = user32.NewProc("SetThreadDpiAwarenessContext")
	setWindowComposition      = user32.NewProc("SetWindowCompositionAttribute")
	trackPopupMenuEx          = user32.NewProc("TrackPopupMenuEx")
	findWindowW               = user32.NewProc("FindWindowW")
	findWindowExW             = user32.NewProc("FindWindowExW")
	getCursorPos              = user32.NewProc("GetCursorPos")
	createPopupMenu           = user32.NewProc("CreatePopupMenu")
	appendMenuW               = user32.NewProc("AppendMenuW")
	destroyMenu               = user32.NewProc("DestroyMenu")
	checkMenuItem             = user32.NewProc("CheckMenuItem")
	setForegroundWindow       = user32.NewProc("SetForegroundWindow")
	postMessageW              = user32.NewProc("PostMessageW")
	messageBoxW               = user32.NewProc("MessageBoxW")
	registerWindowMessageW    = user32.NewProc("RegisterWindowMessageW")

	createMutexW     = kernel32.NewProc("CreateMutexW")
	closeHandle      = kernel32.NewProc("CloseHandle")
	getLastError     = kernel32.NewProc("GetLastError")
	getModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	regOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	regSetValueExW   = advapi32.NewProc("RegSetValueExW")
	regDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	regQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	regCloseKey      = advapi32.NewProc("RegCloseKey")

	shellNotifyIconW       = shell32.NewProc("Shell_NotifyIconW")
	setCurrentProcessAppID = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")

	gdiplusStartup            = gdiPlus.NewProc("GdiplusStartup")
	gdiplusShutdown           = gdiPlus.NewProc("GdiplusShutdown")
	gdipCreateBitmapFromFile  = gdiPlus.NewProc("GdipCreateBitmapFromFile")
	gdipCreateHICONFromBitmap = gdiPlus.NewProc("GdipCreateHICONFromBitmap")
	gdipDisposeImage          = gdiPlus.NewProc("GdipDisposeImage")
)
