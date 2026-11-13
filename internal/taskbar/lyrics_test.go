package taskbar

import (
	"image"
	"image/color"
	"testing"
)

// TestLayoutView puts the cover at the left, centred, and the lines after
// it, the line sung now over the next, scaled by the DPI.
func TestLayoutView(t *testing.T) {
	l := layoutView(360, 48, 96, true)
	if l.cover != image.Rect(8, 7, 42, 41) || l.radius != 6 {
		t.Errorf("cover at %v, radius %d", l.cover, l.radius)
	}
	if l.current != (rect{50, 7, 356, 24}) || l.next != (rect{50, 24, 356, 41}) {
		t.Errorf("lines at %v and %v", l.current, l.next)
	}
	if l.currentPx != 13 || l.nextPx != 12 {
		t.Errorf("fonts %d and %d", l.currentPx, l.nextPx)
	}
	l = layoutView(720, 96, 192, false)
	if !l.cover.Empty() || l.current.left != 16 || l.currentPx != 26 {
		t.Errorf("at 200 %%, without a cover: %+v", l)
	}
}

// TestRoundCover leaves the corners clear, the middle and the edges'
// middles solid.
func TestRoundCover(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	c := roundCover(src, 34, 6)
	for _, p := range []image.Point{{0, 0}, {33, 0}, {0, 33}, {33, 33}} {
		if a := c.NRGBAAt(p.X, p.Y).A; a != 0 {
			t.Errorf("corner %v: alpha %d", p, a)
		}
	}
	for _, p := range []image.Point{{17, 17}, {17, 0}, {0, 17}, {33, 17}} {
		if a := c.NRGBAAt(p.X, p.Y).A; a != 255 {
			t.Errorf("%v: alpha %d", p, a)
		}
	}
}

// TestOver puts an opaque pixel in place, and half of one over another.
func TestOver(t *testing.T) {
	if got := over(0x80808080, color.NRGBA{0x10, 0x20, 0x30, 255}); got != 0xFF102030 {
		t.Errorf("opaque: %08x", got)
	}
	if got := over(0, color.NRGBA{255, 0, 0, 128}); got != 0x80800000 {
		t.Errorf("half: %08x", got)
	}
}
