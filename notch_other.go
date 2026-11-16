//go:build !darwin

package main

import "github.com/egoist/mygo"

// findHousing finds the camera housing of a MacBook's screen: none but on
// macOS.
func findHousing() (housing, bool) { return housing{}, false }

// trackPointer does nothing but on macOS, where a window hears the pointer
// only while its app is active.
func trackPointer(*mygo.Window) {}

// newNotchHost does nothing but on macOS, where alone the window around
// the notch opens.
func newNotchHost(*mygo.Window) notchHost { return noNotchHost{} }

type noNotchHost struct{}

func (noNotchHost) keyboard(bool) {}
func (noNotchHost) close()        {}
