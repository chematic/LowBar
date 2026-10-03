//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func scheduleDelayedFileReplace(source, destination string) error {
	src, err := syscall.UTF16FromString(source)
	if err != nil {
		return err
	}
	dst, err := syscall.UTF16FromString(destination)
	if err != nil {
		return err
	}
	ret, _, callErr := moveFileExW.Call(
		uintptr(unsafe.Pointer(&src[0])),
		uintptr(unsafe.Pointer(&dst[0])),
		moveFileReplaceExisting|moveFileDelayUntilReboot,
	)
	if ret == 0 {
		if callErr != nil {
			return callErr
		}
		return fmt.Errorf("MoveFileExW failed")
	}
	return nil
}
