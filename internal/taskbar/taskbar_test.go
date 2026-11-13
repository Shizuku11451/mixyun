package taskbar

import (
	"image/color"
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

// TestEmbedPlace puts the window left of the notification area, scaled by
// the DPI, and nowhere on a taskbar at a side or too short for it.
func TestEmbedPlace(t *testing.T) {
	bar := rect{0, 1032, 1920, 1080}
	for name, c := range map[string]struct {
		bar, tray rect
		dpi       uint32
		x, w, h   int32
		ok        bool
	}{
		"tray":      {bar, rect{1600, 1032, 1920, 1080}, 96, 1240, 360, 48, true},
		"hidpi":     {bar, rect{1600, 1032, 1920, 1080}, 144, 1060, 540, 48, true},
		"no tray":   {bar, rect{}, 0, 1560, 360, 48, true},
		"top":       {rect{0, 0, 1920, 48}, rect{1700, 0, 1920, 48}, 96, 1340, 360, 48, true},
		"side":      {rect{0, 0, 48, 1080}, rect{}, 96, 0, 0, 0, false},
		"too short": {rect{0, 1032, 500, 1080}, rect{300, 1032, 500, 1080}, 96, -60, 360, 48, false},
	} {
		x, w, h, ok := embedPlace(c.bar, c.tray, 360, c.dpi)
		if ok != c.ok || ok && (x != c.x || w != c.w || h != c.h) {
			t.Errorf("%s: %d %d %d %v, want %d %d %d %v", name, x, w, h, ok, c.x, c.w, c.h, c.ok)
		}
	}
}
