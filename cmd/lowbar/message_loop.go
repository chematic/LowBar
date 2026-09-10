//go:build windows

package main

import "unsafe"

func messageLoop() int {
	var message msg
	for {
		result, _, err := getMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			logWin32Error("message.loop", "GetMessageW", err)
			return 1
		}
		if result == 0 {
			return 0
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}
