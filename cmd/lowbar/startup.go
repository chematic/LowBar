//go:build windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func setStartup(enabled bool) error {
	keyPath := utf16ptr("Software\\Microsoft\\Windows\\CurrentVersion\\Run")
	valueName := utf16ptr(appName)
	var key syscall.Handle
	ret, _, _ := regOpenKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(keyPath)), 0, keyWrite, uintptr(unsafe.Pointer(&key)))
	if ret != errorSuccess {
		return syscall.Errno(ret)
	}
	defer regCloseKey.Call(uintptr(key))
	if !enabled {
		ret, _, _ = regDeleteValueW.Call(uintptr(key), uintptr(unsafe.Pointer(valueName)))
		if ret == errorFileNotFound {
			return nil
		}
		if ret != errorSuccess {
			return syscall.Errno(ret)
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	value, err := syscall.UTF16FromString("\"" + exe + "\"")
	if err != nil {
		return err
	}
	valueBytes := unsafe.Slice((*byte)(unsafe.Pointer(&value[0])), len(value)*2)
	ret, _, _ = regSetValueExW.Call(uintptr(key), uintptr(unsafe.Pointer(valueName)), 0, regSz, uintptr(unsafe.Pointer(&valueBytes[0])), uintptr(len(valueBytes)))
	if ret != errorSuccess {
		return syscall.Errno(ret)
	}
	return nil
}

func isStartupEnabled() bool {
	keyPath := utf16ptr("Software\\Microsoft\\Windows\\CurrentVersion\\Run")
	valueName := utf16ptr(appName)
	var key syscall.Handle
	ret, _, _ := regOpenKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(keyPath)), 0, keyRead, uintptr(unsafe.Pointer(&key)))
	if ret != errorSuccess {
		return false
	}
	defer regCloseKey.Call(uintptr(key))
	var valueType uint32
	var data [2048]byte
	size := uint32(len(data))
	ret, _, _ = regQueryValueExW.Call(uintptr(key), uintptr(unsafe.Pointer(valueName)), 0, uintptr(unsafe.Pointer(&valueType)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&size)))
	return ret == errorSuccess && size > 0
}
