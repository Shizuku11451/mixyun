// Package taskbar puts the player's buttons on the thumbnail of a window's
// taskbar button, which Windows shows while the pointer rests on the
// button: previous, play or pause, and next, as music players have them
// there.
package taskbar

import (
	"cmp"
	"errors"
	"image"
	"image/color"
	"slices"
)

// ErrUnsupported is returned where windows have no thumbnail buttons.
var ErrUnsupported = errors.New("taskbar: thumbnail buttons are not supported here")

// Button is a button of the thumbnail.
type Button int

const (
	Previous Button = iota
	PlayPause
	Next
)

// Bar is the buttons on a window's thumbnail.
type Bar struct{ b *bar }

// Attach puts the buttons on the thumbnail of the window hwnd, and has
// click take their clicks. It runs on the window's thread, which click runs
// on too. The buttons show a player that doesn't play, with nothing to
// play, until Show.
func Attach(hwnd uintptr, click func(Button)) (*Bar, error) {
	b, err := attach(hwnd, click)
	if err != nil {
		return nil, err
	}
	return &Bar{b}, nil
}

// Show shows whether the player plays, and whether it has a track: without
// one, the buttons are off. It returns at once, from any goroutine; the
// buttons go with their window.
func (b *Bar) Show(playing, track bool) { b.b.show(state{playing, track}) }

// Lyrics is the lyrics sung now at the left end of the primary taskbar,
// with the track's cover.
type Lyrics struct{ l *lyricsWin }

// OpenLyrics puts a window width DIPs wide in the primary taskbar, as its
// child, so that it hides and slides with it and nothing comes over it,
// clear of the taskbar's buttons, narrower where they leave less room. It
// shows nothing until Show. Its cover shows a button that expands under
// the pointer, and a click on it calls expand, on the window's own thread;
// the taskbar under the rest of it takes the clicks. It returns
// ErrUnsupported but on Windows, and an error where the taskbar takes no
// window, as one at a side.
func OpenLyrics(width int, expand func()) (*Lyrics, error) {
	l, err := openLyrics(width, expand)
	if err != nil {
		return nil, err
	}
	return &Lyrics{l}, nil
}

// Show shows v, in the taskbar's ink, or hides the window without
// v.Current. It returns at once, from any goroutine.
func (l *Lyrics) Show(v LyricsView) { l.l.show(v) }

// Close closes the window, from any goroutine.
func (l *Lyrics) Close() { l.l.close() }

// Bounds is where the window is on the screen, in pixels, while it shows;
// empty while it doesn't. From any goroutine.
func (l *Lyrics) Bounds() image.Rectangle { return l.l.bounds() }

// Closed tells whether the window is gone: closed, or gone with the
// taskbar, as Explorer restarts. Another OpenLyrics puts it back.
func (l *Lyrics) Closed() bool { return l.l.isClosed() }

// inkPixels turns what GDI drew white on black, in the BGRA pixels px, into
// the premultiplied pixels of ink that UpdateLayeredWindow takes: the grey
// of a pixel is its alpha.
func inkPixels(px []uint32, ink [3]uint8) {
	for i, p := range px {
		a := max(p&0xFF, p>>8&0xFF, p>>16&0xFF)
		if a == 0 {
			px[i] = 0
			continue
		}
		r, g, b := uint32(ink[0])*a/255, uint32(ink[1])*a/255, uint32(ink[2])*a/255
		px[i] = a<<24 | r<<16 | g<<8 | b
	}
}

// rect is a window's bounds, as RECT has them.
type rect struct{ left, top, right, bottom int32 }

// embedPlace is where embed puts a window width DIPs wide in the taskbar
// bar at dpi, clear of its buttons, on the screen as UI Automation has
// them: in the leftmost gap that takes it whole, or else narrowed into the
// widest, down to half its width, or else at its left end, over them.
// Windows 11 leaves one between the
// widgets and its centred buttons; a taskbar whose buttons stand at the
// left leaves one after them. It returns x in the taskbar, the width and
// the height, in pixels; ok is false for a taskbar at a side.
func embedPlace(bar rect, buttons []rect, width int, dpi uint32) (x, w, h int32, ok bool) {
	if bar.bottom-bar.top >= bar.right-bar.left {
		return 0, 0, 0, false
	}
	if dpi == 0 {
		dpi = 96
	}
	w, h = int32(width)*int32(dpi)/96, bar.bottom-bar.top
	pad, end := 8*int32(dpi)/96, bar.right-bar.left
	// What the buttons take of the bar, from its left, padded.
	var taken [][2]int32
	for _, b := range buttons {
		if b.right <= b.left || b.bottom <= bar.top || b.top >= bar.bottom {
			continue // hidden, or on another taskbar
		}
		if l, r := max(b.left-pad-bar.left, 0), min(b.right+pad-bar.left, end); l < r {
			taken = append(taken, [2]int32{l, r})
		}
	}
	slices.SortFunc(taken, func(a, b [2]int32) int { return cmp.Compare(a[0], b[0]) })
	from, bestX, bestW := int32(0), int32(0), int32(0)
	for _, t := range append(taken, [2]int32{end, end}) {
		if gap := t[0] - from; gap >= w {
			return from, w, h, true
		} else if gap > bestW {
			bestX, bestW = from, gap
		}
		from = max(from, t[1])
	}
	if bestW > 0 && bestW >= w/2 {
		return bestX, bestW, h, true
	}
	// No room between them, as the buttons read: at the left end, over
	// them, rather than no lyrics, should the reading be wrong.
	return 0, min(w, end), h, true
}

// state is what the buttons show: whether the player plays, and whether it
// has a track, without which they're off.
type state struct{ playing, track bool }

// point is a point of a glyph, in a square of side 1, y down.
type point struct{ x, y float64 }

// glyph is the convex shapes an icon is made of.
type glyph [][]point

// The glyphs of the buttons, as the player bar draws them.
var (
	glyphPrevious = glyph{
		{{0.22, 0.25}, {0.32, 0.25}, {0.32, 0.75}, {0.22, 0.75}},
		{{0.80, 0.25}, {0.80, 0.75}, {0.36, 0.50}},
	}
	glyphPlay  = glyph{{{0.30, 0.20}, {0.80, 0.50}, {0.30, 0.80}}}
	glyphPause = glyph{
		{{0.26, 0.22}, {0.42, 0.22}, {0.42, 0.78}, {0.26, 0.78}},
		{{0.58, 0.22}, {0.74, 0.22}, {0.74, 0.78}, {0.58, 0.78}},
	}
	// glyphExpand is two chevrons up, which open the player over the
	// taskbar's lyrics.
	glyphExpand = glyph{
		{{0.24, 0.48}, {0.50, 0.24}, {0.50, 0.36}, {0.24, 0.60}},
		{{0.50, 0.24}, {0.76, 0.48}, {0.76, 0.60}, {0.50, 0.36}},
		{{0.24, 0.72}, {0.50, 0.48}, {0.50, 0.60}, {0.24, 0.84}},
		{{0.50, 0.48}, {0.76, 0.72}, {0.76, 0.84}, {0.50, 0.60}},
	}
	glyphNext = glyph{
		{{0.20, 0.25}, {0.64, 0.50}, {0.20, 0.75}},
		{{0.68, 0.25}, {0.78, 0.25}, {0.78, 0.75}, {0.68, 0.75}},
	}
)

// samples is how many samples a pixel's side takes: their share inside the
// glyph is the pixel's alpha, which smooths the edges.
const samples = 4

// render draws g in ink, size pixels square.
func (g glyph) render(size int, ink color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			in := 0
			for sy := range samples {
				for sx := range samples {
					p := point{(float64(x) + (float64(sx)+0.5)/samples) / float64(size), (float64(y) + (float64(sy)+0.5)/samples) / float64(size)}
					if g.covers(p) {
						in++
					}
				}
			}
			if in > 0 {
				c := ink
				c.A = uint8(int(ink.A) * in / (samples * samples))
				img.SetNRGBA(x, y, c)
			}
		}
	}
	return img
}

// covers tells whether p is inside one of the glyph's shapes.
func (g glyph) covers(p point) bool {
	for _, shape := range g {
		if inside(shape, p) {
			return true
		}
	}
	return false
}

// inside tells whether p is inside the convex polygon shape: on the same
// side of all of its edges.
func inside(shape []point, p point) bool {
	var pos, neg bool
	for i, a := range shape {
		b := shape[(i+1)%len(shape)]
		cross := (b.x-a.x)*(p.y-a.y) - (b.y-a.y)*(p.x-a.x)
		pos = pos || cross > 0
		neg = neg || cross < 0
	}
	return !(pos && neg)
}
