//go:build !windows

package main

import "github.com/egoist/mygo"

// floatVibrancy is the system's glass: Liquid Glass on macOS 26, the
// closest material on older macOS, none on Linux.
const floatVibrancy = mygo.VibrancyGlass

// setFloatGlass shows the system's glass behind the window w, or none. It
// reports false: the system draws the glass and its tint itself. Main
// thread.
func setFloatGlass(w *mygo.Window, on bool) bool {
	v := mygo.VibrancyNone
	if on {
		v = floatVibrancy
	}
	w.SetVibrancy(v)
	return false
}
