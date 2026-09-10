//go:build windows

package main

import "syscall"

func utf16ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func utf16from(s string, dst []uint16) {
	v, _ := syscall.UTF16FromString(s)
	copy(dst, v)
}

func makeConfig() config {
	return config{style: styleBlur, language: langEnglish, startup: false}
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
