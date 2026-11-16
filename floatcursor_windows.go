//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"github.com/egoist/mygo"
)

var (
	floatUser32         = syscall.NewLazyDLL("user32.dll")
	procGetCursorPos    = floatUser32.NewProc("GetCursorPos")
	procScreenToClient  = floatUser32.NewProc("ScreenToClient")
	procGetClientRect   = floatUser32.NewProc("GetClientRect")
	procGetDpiForWindow = floatUser32.NewProc("GetDpiForWindow")
)

// cursorIn returns where the pointer is in the window w, in DIPs from the
// top left of its client area, and whether it is over it. Main thread.
func cursorIn(w *mygo.Window) (x, y float32, in bool) {
	hwnd := w.NativeHandle()
	if hwnd == 0 {
		return 0, 0, false
	}
	var p struct{ X, Y int32 }
	if ok, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p))); ok == 0 {
		return 0, 0, false
	}
	procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&p)))
	var r struct{ Left, Top, Right, Bottom int32 }
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	scale := float32(1)
	if dpi, _, _ := procGetDpiForWindow.Call(hwnd); dpi > 0 {
		scale = float32(dpi) / 96
	}
	in = p.X >= 0 && p.Y >= 0 && p.X < r.Right && p.Y < r.Bottom
	return float32(p.X) / scale, float32(p.Y) / scale, in
}
