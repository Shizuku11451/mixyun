package taskbar

import (
	"image"
	"image/color"
	"testing"
	"time"
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

// TestGlide eases from 0 to 1, and mixes pixels a share of the way.
func TestGlide(t *testing.T) {
	if glideEase(0) != 0 || glideEase(1) != 1 || glideEase(0.5) <= 0.5 {
		t.Errorf("ease: %v %v %v", glideEase(0), glideEase(0.5), glideEase(1))
	}
	px := []uint32{0xFFFFFFFF, 0}
	mixPixels(px, []uint32{0, 0xFF804000}, 0.5)
	if px[0] != 0x7F7F7F7F || px[1] != 0x7F402000 {
		t.Errorf("mix: %08x %08x", px[0], px[1])
	}
}

// TestSweep lights the words sung, the one sung now in part, and dims the
// pixels past the edge.
func TestSweep(t *testing.T) {
	words := []Word{{"a", time.Second, 2 * time.Second}, {"b", 3 * time.Second, 5 * time.Second}}
	spans := []span{{0, 10}, {12, 32}}
	for at, want := range map[time.Duration]float64{0: 0, 1500 * time.Millisecond: 5, 2500 * time.Millisecond: 10, 4 * time.Second: 22, 9 * time.Second: 32} {
		if got := sweepAt(words, spans, at); got != want {
			t.Errorf("at %v: %v, want %v", at, got, want)
		}
	}
	px := []uint32{0xFFFFFF, 0xFFFFFF, 0xFFFFFF, 0xFFFFFF}
	sweepPixels(px, 4, rect{0, 0, 4, 1}, 2, 1, 1, 0.5)
	if px[0] != 0xFFFFFF || px[3] != 0x7F7F7F || px[1] != 0xFFFFFF || px[2] != 0x7F7F7F {
		t.Errorf("sweep: %06x", px)
	}
	v := LyricsView{Song: time.Second, Seen: time.Unix(10, 0), Speed: 2}
	if got := v.songAt(time.Unix(10, 0).Add(time.Second)); got != 3*time.Second {
		t.Errorf("song at: %v", got)
	}
}

// TestHoverCover darkens the cover and puts the chevrons in its middle.
func TestHoverCover(t *testing.T) {
	c := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for i := range c.Pix {
		c.Pix[i] = 200
	}
	hoverCover(c)
	if got := c.NRGBAAt(2, 2); got.R != 90 || got.A != 200 {
		t.Errorf("corner: %v", got)
	}
	if got := c.NRGBAAt(20, 15); got.R < 200 {
		t.Errorf("chevron: %v", got)
	}
}
