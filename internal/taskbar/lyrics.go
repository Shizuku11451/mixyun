package taskbar

// What the lyrics in the taskbar look like, as TaskbarLyrics has them: the
// cover, a rounded square, then the line sung now over the next one,
// dimmer. Sizes are in DIPs, at 96 DPI.

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/draw"
)

// LyricsView is what the lyrics in the taskbar show.
type LyricsView struct {
	// Current is the line sung now, and Next the one after it, "" for
	// none. A view without Current hides the window.
	Current, Next string
	// Cover is the track's cover, cropped square; nil shows none.
	Cover image.Image
}

const (
	viewCoverSize   = 34
	viewCoverRadius = 6
	viewCoverGap    = 8
	viewPadLeft     = 8
	viewPadRight    = 4
	// The lines' sizes, and how far apart their middles are.
	viewCurrentSize = 13
	viewNextSize    = 12
	viewLinePitch   = 17
	// The lines' opacity, as the ink's share.
	viewCurrentAlpha = 0.90
	viewNextAlpha    = 0.60
)

// viewLayout is where the parts of a view go in a window w by h pixels.
type viewLayout struct {
	cover         image.Rectangle // empty without a cover
	radius        int
	current, next rect
	currentPx     int32 // font heights
	nextPx        int32
}

// layoutView lays a view out in a window w by h pixels, at dpi.
func layoutView(w, h int32, dpi uint32, cover bool) viewLayout {
	if dpi == 0 {
		dpi = 96
	}
	px := func(dips float64) int32 { return int32(math.Round(dips * float64(dpi) / 96)) }
	var l viewLayout
	x := px(viewPadLeft)
	if cover {
		side := min(px(viewCoverSize), h)
		y := (h - side) / 2
		l.cover = image.Rect(int(x), int(y), int(x+side), int(y+side))
		l.radius = int(px(viewCoverRadius))
		x += side + px(viewCoverGap)
	}
	// Each line is centred in a row a pitch tall, the two rows about the
	// window's middle.
	mid, pitch, right := h/2, px(viewLinePitch), w-px(viewPadRight)
	l.current = rect{x, mid - pitch, right, mid}
	l.next = rect{x, mid, right, mid + pitch}
	l.currentPx, l.nextPx = px(viewCurrentSize), px(viewNextSize)
	return l
}

// roundCover is img scaled to side pixels square, its corners rounded at
// radius, smooth at their edges.
func roundCover(img image.Image, side, radius int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, side, side))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
	r := float64(min(radius, side/2))
	for y := range side {
		for x := range side {
			// The distance past the arc of the corner the pixel is in.
			cx := math.Max(r-float64(x)-0.5, float64(x)+0.5-(float64(side)-r))
			cy := math.Max(r-float64(y)-0.5, float64(y)+0.5-(float64(side)-r))
			if cx <= 0 || cy <= 0 {
				continue
			}
			cover := math.Min(1, math.Max(0, r-math.Hypot(cx, cy)+0.5))
			c := dst.NRGBAAt(x, y)
			c.A = uint8(float64(c.A)*cover + 0.5)
			dst.SetNRGBA(x, y, c)
		}
	}
	return dst
}

// pasteCover puts the cover over the premultiplied BGRA pixels px of a
// window stride pixels wide, at its top left corner at.
func pasteCover(px []uint32, stride int, at image.Point, cover *image.NRGBA) {
	b := cover.Bounds()
	for y := range b.Dy() {
		for x := range b.Dx() {
			i := (at.Y+y)*stride + at.X + x
			if i < 0 || i >= len(px) || at.X+x >= stride {
				continue
			}
			c := cover.NRGBAAt(x, y)
			px[i] = over(px[i], c)
		}
	}
}

// over is the premultiplied BGRA pixel dst under c.
func over(dst uint32, c color.NRGBA) uint32 {
	a := uint32(c.A)
	inv := 255 - a
	ch := func(src uint8, shift uint) uint32 {
		return (uint32(src)*a + (dst>>shift&0xFF)*inv + 127) / 255
	}
	return ch(255, 24)<<24 | ch(c.R, 16)<<16 | ch(c.G, 8)<<8 | ch(c.B, 0)
}
