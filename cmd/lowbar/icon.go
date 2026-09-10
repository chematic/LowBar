//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func iconPath() string {
	e, err := os.Executable()
	if err != nil {
		return filepath.Join("Assets", "icon.png")
	}
	return filepath.Join(filepath.Dir(e), "Assets", "icon.png")
}

func loadPNGIcon(path string) (syscall.Handle, error) {
	var token uintptr
	input := gdiplusStartupInput{gdiplusVersion: 1}
	status, _, callErr := gdiplusStartup.Call(uintptr(unsafe.Pointer(&token)), uintptr(unsafe.Pointer(&input)), 0)
	if status != 0 {
		if callErr != nil {
			return 0, callErr
		}
		return 0, fmt.Errorf("GDI+ startup failed: status %d", status)
	}
	defer gdiplusShutdown.Call(token)
	var bitmap uintptr
	utfPath := utf16ptr(path)
	status, _, callErr = gdipCreateBitmapFromFile.Call(uintptr(unsafe.Pointer(utfPath)), uintptr(unsafe.Pointer(&bitmap)))
	if status != 0 || bitmap == 0 {
		if callErr != nil {
			return 0, callErr
		}
		return 0, fmt.Errorf("could not load icon: %s", path)
	}
	defer gdipDisposeImage.Call(bitmap)
	var hicon uintptr
	status, _, callErr = gdipCreateHICONFromBitmap.Call(bitmap, uintptr(unsafe.Pointer(&hicon)))
	if status != 0 || hicon == 0 {
		if callErr != nil {
			return 0, callErr
		}
		return 0, fmt.Errorf("could not create icon from PNG: %s", path)
	}
	return syscall.Handle(hicon), nil
}

func loadTrayIcon() error {
	path := iconPath()
	icon, err := loadPNGIcon(path)
	if err == nil {
		trayIcon = icon
		customTrayIcon = true
		return nil
	}
	logLine("custom icon unavailable: " + err.Error())
	fallback, _, fallbackErr := loadIconW.Call(0, iconApplication)
	if fallback == 0 {
		if fallbackErr != nil {
			return fallbackErr
		}
		return err
	}
	trayIcon = syscall.Handle(fallback)
	customTrayIcon = false
	return nil
}

func destroyTrayIconHandle() {
	if trayIcon == 0 || !customTrayIcon {
		trayIcon = 0
		customTrayIcon = false
		return
	}
	user32.NewProc("DestroyIcon").Call(uintptr(trayIcon))
	trayIcon = 0
	customTrayIcon = false
}
