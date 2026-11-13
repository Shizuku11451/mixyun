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

// Light tells whether the taskbar is light, as Windows' theme of its own
// parts has it: false elsewhere.
func Light() bool { return light() }

// KeepAbove puts the topmost window hwnd back above the taskbar. It does
// nothing but on Windows.
func KeepAbove(hwnd uintptr) { keepAbove(hwnd) }

// Embed makes the window hwnd a child of the primary taskbar, as tall as
// it, width DIPs wide, left of its notification area, or moves it there
// again: the taskbar carries it, so nothing comes over it, and it hides
// and slides with it. It returns false, leaving hwnd as it was, where
// there is no such taskbar, or it stands at a side: always but on Windows.
func Embed(hwnd uintptr, width int) bool { return embed(hwnd, width) }

// rect is a window's bounds, as RECT has them.
type rect struct{ left, top, right, bottom int32 }

// embedPlace is where Embed puts a window width DIPs wide in the taskbar
// bar, whose notification area is tray, at dpi: x in the taskbar, its
// width and height, in pixels. ok is false for a taskbar at a side, or
// too short for the window.
// shortcut: only the notification area is measured, not the taskbar's
// buttons, which a full taskbar puts under the lyrics; read them through UI
// Automation, as TaskbarLyrics does, if that bothers.
func embedPlace(bar, tray rect, width int, dpi uint32) (x, w, h int32, ok bool) {
	if bar.bottom-bar.top >= bar.right-bar.left {
		return 0, 0, 0, false
	}
	if dpi == 0 {
		dpi = 96
	}
	right := bar.right
	if tray.right > tray.left && tray.left > bar.left && tray.left <= bar.right {
		right = tray.left
	}
	w = int32(width) * int32(dpi) / 96
	x = right - bar.left - w
	return x, w, bar.bottom - bar.top, x >= 0
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
