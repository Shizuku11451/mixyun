package taskbar

import (
	"image/color"
	"slices"
	"testing"
)

// TestGlyphs draws each glyph solid where its shapes are, clear around
// them, and smooth at their edges.
func TestGlyphs(t *testing.T) {
	const size = 32
	white := color.NRGBA{255, 255, 255, 255}
	for name, c := range map[string]struct {
		g       glyph
		in, out []point
	}{
		"previous": {glyphPrevious, []point{{0.27, 0.5}, {0.6, 0.5}}, []point{{0.1, 0.1}, {0.9, 0.5}, {0.35, 0.3}}},
		"play":     {glyphPlay, []point{{0.4, 0.5}, {0.7, 0.5}}, []point{{0.1, 0.1}, {0.85, 0.5}, {0.5, 0.25}}},
		"pause":    {glyphPause, []point{{0.34, 0.5}, {0.66, 0.5}}, []point{{0.5, 0.5}, {0.1, 0.1}}},
		"next":     {glyphNext, []point{{0.4, 0.5}, {0.73, 0.5}}, []point{{0.9, 0.9}, {0.1, 0.5}, {0.62, 0.3}}},
	} {
		img := c.g.render(size, white)
		alpha := func(p point) uint8 { return img.NRGBAAt(int(p.x*size), int(p.y*size)).A }
		for _, p := range c.in {
			if a := alpha(p); a != 255 {
				t.Errorf("%s: alpha %d at %v, inside", name, a, p)
			}
		}
		for _, p := range c.out {
			if a := alpha(p); a != 0 {
				t.Errorf("%s: alpha %d at %v, outside", name, a, p)
			}
		}
		smooth := false
		for i := 3; i < len(img.Pix); i += 4 {
			if a := img.Pix[i]; a > 0 && a < 255 {
				smooth = true
				if c := img.Pix[i-3]; c != 255 {
					t.Fatalf("%s: a pixel partly in is %d, not white with less alpha", name, c)
				}
			}
		}
		if !smooth {
			t.Errorf("%s: no pixel is partly in: the edges are jagged", name)
		}
	}
}

// TestEmbedPlace puts the window in the leftmost gap of the taskbar's
// buttons that takes it, scaled by the DPI, else narrowed into the widest,
// and nowhere on a taskbar at a side or without room.
func TestEmbedPlace(t *testing.T) {
	bottom := rect{0, 1032, 1920, 1080}
	widgets := rect{0, 1032, 160, 1080}
	centred := rect{760, 1032, 1160, 1080}
	tray := rect{1700, 1032, 1920, 1080}
	for name, c := range map[string]struct {
		bar     rect
		buttons []rect
		dpi     uint32
		x, w, h int32
		ok      bool
	}{
		"bottom":       {bottom, nil, 96, 0, 360, 48, true},
		"hidpi":        {bottom, nil, 144, 0, 540, 48, true},
		"no dpi":       {bottom, nil, 0, 0, 360, 48, true},
		"top":          {rect{0, 0, 1920, 48}, nil, 96, 0, 360, 48, true},
		"side":         {rect{0, 0, 48, 1080}, nil, 96, 0, 0, 0, false},
		"too short":    {rect{0, 1032, 300, 1080}, nil, 96, 0, 300, 48, true},
		"widgets":      {bottom, []rect{widgets, centred, tray}, 96, 168, 360, 48, true},
		"left buttons": {bottom, []rect{{0, 1032, 600, 1080}, tray}, 96, 608, 360, 48, true},
		"narrowed":     {rect{0, 1032, 1000, 1080}, []rect{widgets, {440, 1032, 1000, 1080}}, 96, 168, 264, 48, true},
		"no room":      {rect{0, 1032, 1000, 1080}, []rect{{0, 1032, 1000, 1080}}, 96, 0, 0, 48, false},
		"other bar":    {bottom, []rect{{0, 0, 600, 48}}, 96, 0, 360, 48, true},
		"hidden":       {bottom, []rect{{0, 0, 0, 0}}, 96, 0, 360, 48, true},
	} {
		x, w, h, ok := embedPlace(c.bar, c.buttons, 360, c.dpi)
		if ok != c.ok || ok && (x != c.x || w != c.w || h != c.h) {
			t.Errorf("%s: %d %d %d %v, want %d %d %d %v", name, x, w, h, ok, c.x, c.w, c.h, c.ok)
		}
	}
}

// TestInkPixels makes GDI's grey the alpha of the ink, premultiplied, and
// leaves the rest clear.
func TestInkPixels(t *testing.T) {
	px := []uint32{0, 0x00FFFFFF, 0x00808080, 0x00102030}
	inkPixels(px, [3]uint8{255, 255, 255})
	want := []uint32{0, 0xFFFFFFFF, 0x80808080, 0x30303030}
	if !slices.Equal(px, want) {
		t.Errorf("white ink: %08x, not %08x", px, want)
	}
	px = []uint32{0x00FFFFFF, 0x00808080}
	inkPixels(px, [3]uint8{0x1f, 0x1f, 0x1f})
	want = []uint32{0xFF1F1F1F, 0x800F0F0F}
	if !slices.Equal(px, want) {
		t.Errorf("dark ink: %08x, not %08x", px, want)
	}
}
