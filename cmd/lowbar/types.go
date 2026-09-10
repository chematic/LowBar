//go:build windows

package main

import "syscall"

type point struct{ x, y int32 }

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

type msg struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type notifyIconData struct {
	cbSize            uint32
	hWnd              syscall.Handle
	uID               uint32
	uFlags            uint32
	uCallbackMessage  uint32
	hIcon             syscall.Handle
	szTip             [128]uint16
	dwState           uint32
	dwStateMask       uint32
	szInfo            [256]uint16
	uTimeoutOrVersion uint32
	szInfoTitle       [64]uint16
	dwInfoFlags       uint32
	guidItem          [16]byte
	hBalloonIcon      syscall.Handle
}

type accentPolicy struct {
	state     uint32
	flags     uint32
	gradient  uint32
	animation uint32
}

type compositionAttributeData struct {
	attribute uint32
	data      uintptr
	dataSize  uintptr
}

type config struct {
	style    int
	language int
	startup  bool
}

type stringsTable struct {
	style, normal, opaque, clear, blur, acrylic string
	openBoot, refresh, language                 string
	english, french, spanish, german, russian   string
	about, exit, aboutBody, aboutTitle          string
}

type gdiplusStartupInput struct {
	gdiplusVersion           uint32
	debugEventCallback       uintptr
	suppressBackgroundThread int32
	suppressExternalCodecs   int32
}
