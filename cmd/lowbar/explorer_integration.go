//go:build windows

package main

import (
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const explorerHookDLLName = "LowBarExplorerHook.dll"

func startExplorerIntegration(_ syscall.Handle) {
	ensureExplorerHook("startup")
}

func installExplorerHookForTaskbarCreation() {
	ensureExplorerHook("taskbar-created")
}

func explorerHookPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), explorerHookDLLName), nil
}

func currentTaskbarOwner() (uint32, syscall.Handle) {
	for _, hwnd := range taskbarWindows() {
		var pid uint32
		getWindowThreadProcessID.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
		if pid != 0 {
			return pid, hwnd
		}
	}
	return 0, 0
}

func ensureExplorerHook(reason string) {
	defer recoverPanic("explorer.integration")
	if shuttingDown || globalConfig == nil {
		return
	}
	pid, hwnd := currentTaskbarOwner()
	if pid == 0 || hwnd == 0 {
		logEvent("DEBUG", "explorer.integration", fmt.Sprintf("no taskbar owner available reason=%s", reason))
		return
	}
	if explorerPID == pid {
		return
	}
	dllPath, err := explorerHookPath()
	if err != nil {
		logError("explorer.integration", "resolve hook path", err)
		return
	}
	if _, err := os.Stat(dllPath); err != nil {
		logError("explorer.integration", fmt.Sprintf("hook DLL missing path=%s", dllPath), err)
		return
	}
	module, err := injectExplorerDLL(pid, dllPath)
	if err != nil {
		logError("explorer.integration", fmt.Sprintf("inject reason=%s pid=%d", reason, pid), err)
		return
	}
	explorerPID = pid
	explorerModule = module
	logEvent("INFO", "explorer.integration", fmt.Sprintf("Explorer hook installed pid=%d taskbar=%d reason=%s", pid, hwnd, reason))
}

func exportRVA(path, name string) (uint32, error) {
	f, err := pe.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var exportRVA uint32
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		exportRVA = oh.DataDirectory[0].VirtualAddress
	case *pe.OptionalHeader64:
		exportRVA = oh.DataDirectory[0].VirtualAddress
	default:
		return 0, fmt.Errorf("unsupported PE optional header")
	}
	if exportRVA == 0 {
		return 0, fmt.Errorf("DLL has no export directory")
	}
	var sec *pe.Section
	for _, s := range f.Sections {
		if exportRVA >= s.VirtualAddress && exportRVA < s.VirtualAddress+s.VirtualSize {
			sec = s
			break
		}
	}
	if sec == nil {
		return 0, fmt.Errorf("export directory section not found")
	}
	data, err := sec.Data()
	if err != nil {
		return 0, err
	}
	off := exportRVA - sec.VirtualAddress
	if int(off)+40 > len(data) {
		return 0, fmt.Errorf("invalid export directory")
	}
	namesCount := binary.LittleEndian.Uint32(data[off+24:])
	namesRVA := binary.LittleEndian.Uint32(data[off+32:])
	ordsRVA := binary.LittleEndian.Uint32(data[off+36:])
	funcsRVA := binary.LittleEndian.Uint32(data[off+28:])
	readRVA := func(rva uint32, size int) ([]byte, error) {
		for _, s := range f.Sections {
			if rva >= s.VirtualAddress && rva < s.VirtualAddress+s.VirtualSize {
				d, e := s.Data()
				if e != nil {
					return nil, e
				}
				o := rva - s.VirtualAddress
				if int(o)+size > len(d) {
					return nil, fmt.Errorf("invalid export RVA")
				}
				return d[o : o+uint32(size)], nil
			}
		}
		return nil, fmt.Errorf("RVA 0x%x not found", rva)
	}
	for i := uint32(0); i < namesCount; i++ {
		entry, err := readRVA(namesRVA+i*4, 4)
		if err != nil {
			return 0, err
		}
		nameRVA := binary.LittleEndian.Uint32(entry)
		nameBytes, err := readRVA(nameRVA, 1)
		if err != nil {
			return 0, err
		}
		_ = nameBytes
		var buf []byte
		for n := uint32(0); ; n++ {
			b, err := readRVA(nameRVA+n, 1)
			if err != nil {
				return 0, err
			}
			if b[0] == 0 {
				break
			}
			buf = append(buf, b[0])
		}
		if string(buf) != name {
			continue
		}
		ord, err := readRVA(ordsRVA+i*2, 2)
		if err != nil {
			return 0, err
		}
		ordinal := uint32(binary.LittleEndian.Uint16(ord))
		fn, err := readRVA(funcsRVA+ordinal*4, 4)
		if err != nil {
			return 0, err
		}
		return binary.LittleEndian.Uint32(fn), nil
	}
	return 0, fmt.Errorf("export %q not found", name)
}

func stopExplorerHook() error {
	pid := explorerPID
	module := explorerModule
	if pid == 0 || module == 0 {
		return nil
	}
	path, err := explorerHookPath()
	if err != nil {
		return err
	}
	rva, err := exportRVA(path, "LowBarExplorerHookStop")
	if err != nil {
		return err
	}
	process, _, openErr := openProcess.Call(uintptr(processCreateThread|processQueryInformation|processVMOperation|processVMRead), 0, uintptr(pid))
	if process == 0 {
		return fmt.Errorf("OpenProcess for hook stop failed: %w", openErr)
	}
	defer closeHandle.Call(process)
	threadAddr := module + uintptr(rva)
	thread, _, threadErr := createRemoteThread.Call(process, 0, 0, threadAddr, module, 0, 0)
	if thread == 0 {
		return fmt.Errorf("CreateRemoteThread for hook stop failed: %w", threadErr)
	}
	defer closeHandle.Call(thread)
	waitRet, _, waitErr := waitForSingleObject.Call(thread, infinite)
	if waitRet != waitObject0 {
		return fmt.Errorf("WaitForSingleObject for hook stop failed: %w", waitErr)
	}
	explorerPID = 0
	explorerModule = 0
	return nil
}

func injectExplorerDLL(pid uint32, dllPath string) (uintptr, error) {
	access := processCreateThread | processQueryInformation | processVMOperation | processVMRead | processVMWrite
	process, _, openErr := openProcess.Call(uintptr(access), 0, uintptr(pid))
	if process == 0 {
		return 0, fmt.Errorf("OpenProcess failed: %w", openErr)
	}
	defer closeHandle.Call(process)

	pathUTF16, err := syscall.UTF16FromString(dllPath)
	if err != nil {
		return 0, err
	}
	pathBytes := unsafe.Slice((*byte)(unsafe.Pointer(&pathUTF16[0])), len(pathUTF16)*2)
	remote, _, allocErr := virtualAllocEx.Call(process, 0, uintptr(len(pathBytes)), memCommit|memReserve, pageReadWrite)
	if remote == 0 {
		return 0, fmt.Errorf("VirtualAllocEx failed: %w", allocErr)
	}
	defer virtualFreeEx.Call(process, remote, 0, memRelease)

	var written uintptr
	ret, _, writeErr := writeProcessMem.Call(process, remote, uintptr(unsafe.Pointer(&pathBytes[0])), uintptr(len(pathBytes)), uintptr(unsafe.Pointer(&written)))
	if ret == 0 || written != uintptr(len(pathBytes)) {
		return 0, fmt.Errorf("WriteProcessMemory failed: %w", writeErr)
	}

	thread, _, threadErr := createRemoteThread.Call(process, 0, 0, loadLibraryW.Addr(), remote, 0, 0)
	if thread == 0 {
		return 0, fmt.Errorf("CreateRemoteThread failed: %w", threadErr)
	}
	defer closeHandle.Call(thread)

	waitRet, _, waitErr := waitForSingleObject.Call(thread, infinite)
	if waitRet != waitObject0 {
		return 0, fmt.Errorf("WaitForSingleObject failed: %w", waitErr)
	}
	var module uintptr
	ret, _, exitErr := getExitCodeThread.Call(thread, uintptr(unsafe.Pointer(&module)))
	if ret == 0 {
		return 0, fmt.Errorf("GetExitCodeThread failed: %w", exitErr)
	}
	if module == 0 {
		return 0, fmt.Errorf("remote LoadLibraryW failed")
	}
	return uintptr(module), nil
}
