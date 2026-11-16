//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"github.com/egoist/mygo"
)

// floatVibrancy is no system material: Windows draws its Acrylic as a solid
// fill in a window that is not active, as the floating one over another
// app always is, so setFloatGlass blurs behind it instead.
const floatVibrancy = mygo.VibrancyNone

var (
	procSetWindowCompositionAttribute = floatUser32.NewProc("SetWindowCompositionAttribute")
	procFloatDwmSetWindowAttribute    = syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
)

// Windows 11 outlines a window it rounds the corners of: none while the
// window is locked, the lyrics alone over the desktop (DWMWA_BORDER_COLOR).
const (
	dwmwaBorderColor = 34
	dwmColorDefault  = 0xFFFFFFFF
	dwmColorNone     = 0xFFFFFFFE
)

// The accent policy of SetWindowCompositionAttribute, which DWM keeps
// drawing whether the window is active or not (Windows 10 1803 and later).
const (
	wcaAccentPolicy  = 19
	accentDisabled   = 0
	accentBlurBehind = 3
)

type accentPolicy struct{ state, flags, gradient, animation uint32 }

type compositionAttrib struct {
	attrib uint32
	data   unsafe.Pointer
	size   uintptr
}

// setFloatGlass blurs what is behind the window w, a Gaussian blur, and
// outlines it, or stops both, and reports whether it blurs: then the
// window draws its own tint over it (floatView). Main thread.
func setFloatGlass(w *mygo.Window, on bool) bool {
	hwnd := w.NativeHandle()
	if hwnd == 0 {
		return false
	}
	border := uint32(dwmColorNone)
	if on {
		border = dwmColorDefault
	}
	if procFloatDwmSetWindowAttribute.Find() == nil { // Windows 10 ignores it
		procFloatDwmSetWindowAttribute.Call(hwnd, dwmwaBorderColor, uintptr(unsafe.Pointer(&border)), 4)
	}
	if procSetWindowCompositionAttribute.Find() != nil {
		return false
	}
	p := accentPolicy{state: accentDisabled}
	if on {
		p.state = accentBlurBehind
	}
	a := compositionAttrib{attrib: wcaAccentPolicy, data: unsafe.Pointer(&p), size: unsafe.Sizeof(p)}
	ok, _, _ := procSetWindowCompositionAttribute.Call(hwnd, uintptr(unsafe.Pointer(&a)))
	return on && ok != 0
}
