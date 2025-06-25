package main

import (
	"syscall"
)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procShowWindow    = user32.NewProc("ShowWindow")
	procSetForeground = user32.NewProc("SetForegroundWindow")
	SW_RESTORE        = 9
)

func restoreAndActivate(hwnd syscall.Handle) {
	procShowWindow.Call(uintptr(hwnd), uintptr(SW_RESTORE))
	procSetForeground.Call(uintptr(hwnd))
}
