package taskbar

// What the lyrics in the taskbar look like, as TaskbarLyrics has them: the
// cover, a rounded square, then the line sung now over the next one,
// dimmer. Sizes are in DIPs, at 96 DPI.

import (
	"image"
	"image/color"
	"math"
	"slices"
	"time"

	"golang.org/x/image/draw"
)

// LyricsView is what the lyrics in the taskbar show.
type LyricsView struct {
	// Current is the line sung now, and Next the one after it, "" for
	// none. A view without Current hides the window.
	Current, Next string
	// Cover is the track's cover, cropped square; nil shows none.
	Cover image.Image
	// Words are those of Current, for a line timed word by word, which
	// light up as they are sung; nil for a line timed whole.
	Words []Word
	// Song is the song's time at Seen, from which it moves on at Speed;
	// Speed is 0 while the track stalls.
	Song  time.Duration
	Seen  time.Time
	Speed float64
}

// Word is a word of the line sung now, as Current holds it, and when it
// is sung, in the song's time.
type Word struct {
	Text       string
	Start, End time.Duration
}

// Equal tells whether v shows as o does.
func (v LyricsView) Equal(o LyricsView) bool {
	return v.Current == o.Current && v.Next == o.Next && v.Cover == o.Cover && slices.Equal(v.Words, o.Words) &&
		v.Song == o.Song && v.Seen.Equal(o.Seen) && v.Speed == o.Speed
}

// songGlide is how far past Seen the song's time moves on, as the lyrics
// page's: the player tells it every 250 ms, and past that it stalled.
const songGlide = 1500 * time.Millisecond

// songAt is the song's time at now.
func (v LyricsView) songAt(now time.Time) time.Duration {
	d := min(max(now.Sub(v.Seen), 0), songGlide)
	return v.Song + time.Duration(float64(d)*v.Speed)
}

// span is where a word is in its line, in pixels from the line's start.
type span struct{ from, to int32 }

// sweepAt is how far, in pixels from the line's start, the words sung by
// the song's time at reach, the word sung now its part sung. spans are
// those of the first words.
func sweepAt(words []Word, spans []span, at time.Duration) float64 {
	x := 0.0
	for i, s := range spans {
		switch w := words[i]; {
		case at >= w.End:
			x = float64(s.to)
		case at >= w.Start:
			x = float64(s.from) + float64(s.to-s.from)*float64(at-w.Start)/float64(w.End-w.Start)
		}
	}
	return x
}

// sweepPixels sets the greys of the BGRA pixels px of a window stride
// pixels wide, in box, to lit of their level left of x and dim right of
// it, across a soft edge edge pixels wide.
func sweepPixels(px []uint32, stride int, box rect, x, edge, lit, dim float64) {
	top, bottom := max(box.top, 0), min(box.bottom, int32(len(px)/stride))
	for col := max(box.left, 0); col < min(box.right, int32(stride)); col++ {
		f := min(max((x-float64(col)-0.5)/edge+0.5, 0), 1)
		k := uint32((dim+(lit-dim)*f)*256 + 0.5)
		for row := top; row < bottom; row++ {
			i := int(row)*stride + int(col)
			var m uint32
			for s := uint(0); s < 24; s += 8 {
				m |= (px[i] >> s & 0xFF) * k >> 8 << s
			}
			px[i] = m
		}
	}
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
	// viewSweepEdge is how wide the soft edge of the words being sung is.
	viewSweepEdge = 8
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

// hoverCover darkens the cover, and puts over it the button that expands,
// as it shows under the pointer.
func hoverCover(c *image.NRGBA) {
	for i := 0; i < len(c.Pix); i += 4 {
		for j := range 3 {
			c.Pix[i+j] = uint8(uint16(c.Pix[i+j]) * 9 / 20)
		}
	}
	side := c.Bounds().Dx()
	g := glyphExpand.render(side*3/5, color.NRGBA{255, 255, 255, 255})
	at := (side - g.Bounds().Dx()) / 2
	draw.Draw(c, g.Bounds().Add(image.Pt(at, at)), g, image.Point{}, draw.Over)
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

// glideEase is how far, from 0 to 1, a glide p of the way through its time
// has gone: starting softly, then slowing to a stop, so the lines neither
// jump off nor halt.
func glideEase(p float64) float64 {
	if p < 0.5 {
		return 4 * p * p * p
	}
	return 1 - math.Pow(2-2*p, 3)/2
}

// mixPixels puts in px, e of the way from old to it, each channel apart.
func mixPixels(px, old []uint32, e float64) {
	k := uint32(e*256 + 0.5)
	for i, n := range px {
		var m uint32
		for s := uint(0); s < 32; s += 8 {
			m |= ((old[i]>>s&0xFF)*(256-k) + (n>>s&0xFF)*k) >> 8 << s
		}
		px[i] = m
	}
}
