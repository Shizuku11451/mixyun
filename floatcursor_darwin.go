//go:build darwin

package main

import (
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo"
)

// cursorIn returns where the pointer is in the window w, in DIPs from the
// top left of its content, and whether it is over it: AppKit tells it to a
// window that lets the pointer through too. Main thread.
func cursorIn(w *mygo.Window) (x, y float32, in bool) {
	ns := objc.ID(w.NativeHandle())
	if ns == 0 {
		return 0, 0, false
	}
	withPool(func() {
		p := objc.Send[nsPoint](ns, sel("mouseLocationOutsideOfEventStream"))
		b := objc.Send[nsRect](ns.Send(sel("contentView")), sel("bounds"))
		// AppKit counts from the bottom.
		x, y = float32(p.X), float32(b.Size.Height-p.Y)
		in = p.X >= 0 && p.Y >= 0 && p.X < b.Size.Width && p.Y < b.Size.Height
	})
	return x, y, in
}
