package cmd

import (
	"syscall"
	"unsafe"
)

// ownsConsole reports whether this process is the only one attached to its
// console. That is the case when gitshiny.exe is double-clicked in Explorer:
// Windows opens a console just for it and closes it the moment it exits.
func ownsConsole() bool {
	var pids [2]uint32
	n, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleProcessList").
		Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	return n == 1
}
