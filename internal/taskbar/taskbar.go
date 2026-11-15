// Package taskbar puts the player's buttons on the thumbnail of a window's
// taskbar button, which Windows shows while the pointer rests on the
// button: previous, play or pause, and next, as music players have them
// there.
package taskbar

import (
	"errors"
	"image"
	"image/color"
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
// child, so that it hides and slides with it and nothing comes over it. It
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
// bar at dpi: at its left end, where a taskbar with its buttons centred, as
// Windows 11 has it, leaves room. It returns x in the taskbar, the width
// and the height, in pixels; ok is false for a taskbar at a side, or too
// short for the window.
// shortcut: the taskbar's buttons aren't measured, which a taskbar with
// its buttons at the left, or the widgets button, puts under the lyrics;
// read them through UI Automation, as TaskbarLyrics does, if that bothers.
func embedPlace(bar rect, width int, dpi uint32) (x, w, h int32, ok bool) {
	if bar.bottom-bar.top >= bar.right-bar.left {
		return 0, 0, 0, false
	}
	if dpi == 0 {
		dpi = 96
	}
	w = int32(width) * int32(dpi) / 96
	return 0, w, bar.bottom - bar.top, w <= bar.right-bar.left
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
