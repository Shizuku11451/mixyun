//go:build !darwin && !windows

package main

import "github.com/egoist/mygo"

// cursorIn tells nothing on Linux, where a locked floating window shows no
// unlock button: the tray and the player bar's menu unlock it.
// shortcut: no pointer position under GTK; ask X11 or the compositor if Linux users want the button.
func cursorIn(*mygo.Window) (x, y float32, in bool) { return 0, 0, false }
