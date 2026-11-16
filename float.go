package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/locales"
	"github.com/fzn0x/mixmusic/lyrics"
)

// The floating lyrics' window at 100%, and the least it resizes to.
const floatWidth, floatHeight, floatMinWidth, floatMinHeight = 460, 128, 240, 64

// floatHideAfter is how long the progress and the transport stay after
// the pointer leaves the window.
const floatHideAfter = 1500 * time.Millisecond

// floatArtMax is the most DIPs across the cover is shown at: it is fetched
// so once, and drawn smaller as the window shrinks, rather than fetched
// again at each size.
const floatArtMax = 160

// OpenFloat opens the floating lyrics: a small window of Liquid Glass over
// the others, which resizes, with the cover of what plays, the line of its
// lyrics sung now and the one under it, and the song and the transport
// under them, as the window has room for.
func OpenFloat(a *app.App) *mygo.Window {
	registerFonts()
	s := newShell(a, a.GoPlayer())
	bg := s.lk.bg
	s.attach(mygo.NewWindow(mygo.WindowOptions{
		Title:       locales.FloatWindowTitle(),
		Width:       floatWidth,
		Height:      floatHeight,
		MinWidth:    floatMinWidth,
		MinHeight:   floatMinHeight,
		Frameless:   true,
		Transparent: true,
		// The system's glass, frosting the desktop behind the window: the
		// glass plugin bends only what the window draws itself, and this
		// one draws nothing behind its words (macOS 26; older macOS and
		// Windows 11 get their closest material, Linux none).
		Vibrancy: mygo.VibrancyGlass,
		// Tinted with the theme's ground, so that the theme's words read on
		// it whatever the system's appearance.
		// shortcut: the tint stays the theme's of when it opened; reopen it
		// after changing the theme, until the tint follows it.
		VibrancyTint:      fmt.Sprintf("rgba(%d, %d, %d, 0.45)", bg.R, bg.G, bg.B),
		CornerRadius:      22,
		AlwaysOnTop:       true,
		SkipTaskbar:       true,
		DisableMaximize:   true,
		DisableFullScreen: true,
		StateKey:          "float",
		Content:           ui.View(s.floatView),
	}))
	// Over another app the window is never active, and macOS frosts the
	// glass of a window that is not: keep it drawn as active.
	s.win.SetActiveAppearance(true)
	trackPointer(s.win)
	s.follow()
	return s.win
}

// floatLayout is what the floating lyrics show at a window's size: the
// cover where it is wide enough, the line under the one sung where it is
// tall enough, and the song and the transport under them where it is
// taller still.
type floatLayout struct {
	pad, art, size           float32
	lines                    int
	second, bar, meta, times bool
}

// floatLayoutOf is the layout of a window w by h DIPs; bar is whether
// the progress and the transport show over it on hover. The cover gives its
// room up first, so that the progress beside the transport stays long
// enough to seek with, and then the progress's times.
func floatLayoutOf(w, h float32) floatLayout {
	l := floatLayout{pad: 14}
	if h < 90 {
		l.pad = 10
	}
	inner := h - 2*l.pad
	l.bar = h >= 110
	l.second = h >= 84
	room := w - 3*l.pad - 330 // what the column, at its least, leaves the cover
	if !l.bar {
		room = w - 3*l.pad - 200
	}
	if a := min(inner, w*0.3, floatArtMax, room); a >= 40 {
		l.art = a
	}
	column := w - 2*l.pad - l.art
	if l.art > 0 {
		column -= l.pad
	}
	l.size = max(15, min(h*0.16, 34))
	l.lines = 1
	if h >= 170 {
		l.lines = 2
	}
	l.meta = h >= 100
	l.times = column >= 290
	return l
}

// floatView draws the floating lyrics. Dragging it anywhere but its
// buttons moves the window, and its edges resize it; its menu, on a right
// click, plays or pauses, opens MixMusic or closes it.
func (s *shell) floatView(c *ui.Context) {
	s.st = s.p.State()
	s.followArtwork()
	k := &s.lk
	t := k.theme
	t.Background = ui.Transparent
	c.SetTheme(&t)
	l := floatLayoutOf(c.Size())
	cur, ok := s.st.Current()
	title, artists := k.words(locales.MiniRowNothingPlaying()), locales.MiniRowPlaySomething()
	ly := floatLyric{at: -2, first: "♪"}
	if ok {
		title, artists = cur.Track.Title, names(cur.Track.Artists)
		ly = s.floatLines(cur.Track)
	}
	// Where the system draws no glass, as on Linux, the theme's ground.
	ground := k.bg
	if c.Vibrancy() {
		// All but clear: a window's clear pixels let the clicks through,
		// and the whole window should drag.
		ground = ui.RGBA(0, 0, 0, 0.01)
	}
	root := ui.Row(c).Fill().Background(ground).Radius(22).Padding(l.pad).Gap(l.pad).AlignItems(ui.Center).
		TextColor(k.text).Font(k.body).DragWindow()
	root.ContextMenu(func(m *ui.Menu) {
		// The menu plays and pauses too: the transport shows only on hover.
		if _, label := toggleOf(s.st); ok && m.Item(label).Chosen() {
			s.p.Toggle()
		}
		if m.Item(locales.MiniRowOpen()).Chosen() {
			go s.svc.System.ShowMainWindow()
		}
		if m.Item(locales.FloatClose()).Chosen() {
			go s.win.Close()
		}
	})
	// The progress and the transport show while the pointer is over the
	// window, or drags the progress, over the line under the one sung.
	// They hide floatHideAfter after the pointer leaves.
	show := l.bar && (root.Hovered() || s.seeking)
	anim := ui.Smooth
	if !show {
		anim.Delay = floatHideAfter
	}
	bar := root.AnimateTo("bar", b2f(show), anim)
	rise := float32(8)
	if c.Preferences().ReduceMotion {
		rise = 0
	}
	root.Children(func() {
		if l.art > 0 {
			s.floatCover(c, cur.Track, ok, l.art)
		}
		ui.Column(c).Grow(1).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
			if l.meta {
				ui.Row(c).Gap(6).MinWidth(0).Children(func() {
					ui.Text(c, title).FontSize(11).FontWeight(600).SingleLine().Shrink(0).MaxWidth(180)
					ui.Text(c, artists).FontSize(11).TextColor(k.muted).SingleLine().MinWidth(0)
				})
			}
			ui.Column(c).Grow(1).MinHeight(0).Justify(ui.Center).Clip().Children(func() {
				s.floatLyricView(c, cur.Track.ID, ly, l, 1-bar)
			})
			if bar < 0.01 {
				return
			}
			// The progress under the lyrics, the transport at the far end.
			ui.Row(c.Key("float-bar")).Absolute().Left(0).Right(0).Bottom(0).Gap(10).AlignItems(ui.Center).
				Opacity(bar).Translate(0, (1-bar)*rise).Children(func() {
				s.floatProgress(c, ok, l.times)
				s.floatControls(c, cur.Track, ok)
			})
		})
	})
}

// floatLyricView is the line sung now, lit word by word when the lyrics
// time their words, over the line under it, as opaque as under. A new line rises in as the
// last rises out.
func (s *shell) floatLyricView(c *ui.Context, id string, ly floatLyric, l floatLayout, under float32) {
	k := &s.lk
	tr := ui.ElementTransition{
		Animation: &ui.Smooth,
		Position:  true,
		Enter:     &ui.Motion{Y: l.size, Opacity: 0},
		Exit:      &ui.Motion{Y: -l.size, Opacity: 0},
	}
	if c.Preferences().ReduceMotion {
		tr = ui.ElementTransition{Duration: -1}
	}
	ui.Column(c.Key(floatLineKey{id, ly.at})).Gap(2).MinWidth(0).Transition(tr).Children(func() {
		s.lyricsText(c, ly.first, l.size, 700, k.accent, ly.words, 0, 0).MaxLines(l.lines).MinWidth(0)
		if l.second && ly.second != "" {
			ui.Text(c, ly.second).FontSize(max(11, l.size*0.62)).TextColor(k.muted).SingleLine().MinWidth(0).Opacity(under)
		}
	})
}

// floatLineKey tells one line of the floating lyrics from the next, and
// from the same line of another song.
type floatLineKey struct {
	id string
	at int
}

// floatCover is the cover of what plays, size DIPs across.
func (s *shell) floatCover(c *ui.Context, t media.Track, ok bool, size float32) {
	var b *ui.Bitmap
	if ok {
		b = s.art.get(c, s.artAt(t.Artwork, floatArtMax), s.artPixels(floatArtMax))
	}
	art := ui.Box(c.Key("float-art")).Size(size, size).Shrink(0).Radius(min(12, size/5)).Clip()
	if b == nil {
		art.Background(s.lk.surface3)
	}
	art.Children(func() {
		if b != nil {
			ui.Image(c, b).Size(size, size).Fit(ui.Cover)
		}
	})
}

// floatProgress is where the track is, between its time and its length,
// and seeks once the user lets go, as the player bar's progress does. A
// live stream, or a track that loads, has no seeking.
func (s *shell) floatProgress(c *ui.Context, ok, times bool) {
	k := &s.lk
	dur := s.miniDuration()
	if !s.seeking {
		s.seekTo = s.st.Position.Seconds()
	}
	ui.Row(c).Grow(1).Gap(6).AlignItems(ui.Center).MinWidth(0).Children(func() {
		if times {
			ui.Text(c, clock(time.Duration(s.seekTo*float64(time.Second)))).FontSize(10).TextColor(k.muted).FontFeatures("tnum").Shrink(0)
		}
		seek := s.seekSlider(c, &s.seekTo, max(dur.Seconds(), 1)).Grow(1).MinWidth(0).
			Label(locales.PlayerSeek()).Disabled(!ok || s.st.Loading || s.st.Live)
		_, _, held := seek.Dragged()
		if ok && !s.st.Live && !held && (s.seeking || seek.Changed()) {
			s.p.Seek(time.Duration(s.seekTo * float64(time.Second)))
		}
		s.seeking = held
		if times {
			ui.Text(c, clockLength(dur)).FontSize(10).TextColor(k.muted).FontFeatures("tnum").Shrink(0)
		}
	})
}

// floatControls are the transport, with the like button, as the mini
// player has them.
func (s *shell) floatControls(c *ui.Context, t media.Track, ok bool) {
	ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
		if s.miniButton(c, "float-previous", "previous", locales.PlayerPrevious(), false, 30, 18).Clicked() {
			s.p.Previous()
		}
		name, label := toggleOf(s.st)
		toggle := s.round(c, "float-toggle", name, label, true, s.st.Loading || !ok && len(s.st.Queue) == 0, 34, 16)
		if pulses(s.st) {
			toggle.Opacity(0.55 + 0.45*toggle.Loop("busy", 900*time.Millisecond, ui.EaseInOut))
		}
		if toggle.Clicked() {
			s.p.Toggle()
		}
		if s.miniButton(c, "float-next", "next", locales.PlayerNext(), false, 30, 18).Clicked() {
			s.p.Next()
		}
		if ok {
			s.likeButton(c, "float-like", t, 16)
		}
	})
}

// floatLyric is what the floating lyrics show of the lyrics: the line
// sung now, its index (-1 before the first, -2 for a note in place of
// lyrics), its words when the lyrics time them, and the line under it.
type floatLyric struct {
	at            int
	first, second string
	words         []lyrics.Word
}

// floatLines are the lines of lyrics the floating lyrics show of t, from
// the lyrics Catalog.Lyrics keeps (floatText), or why there are none.
func (s *shell) floatLines(t media.Track) floatLyric {
	note := func(text string) floatLyric { return floatLyric{at: -2, first: text} }
	if t.IsEpisode() {
		return note("♪")
	}
	l := fetch(s, "lyrics:"+t.ID, func(ctx context.Context) (lyrics.Lyrics, error) { return s.svc.Catalog.Lyrics(ctx, t) })
	switch {
	case !l.done:
		return note(locales.NotchPanelLyricsLooking())
	case l.err != nil:
		return note("♪")
	case l.value.Instrumental:
		return note(locales.NotchPanelInstrumental())
	case !l.value.Synced || len(l.value.Lines) == 0:
		return note(locales.NotchPanelNoTimed())
	}
	lines := l.value.Lines
	latin, how := s.latinLines(t, l.value)
	f := floatText(lines, latin, how, s.st.Position+lyricsLead(s.st.Speed))
	if f.at >= 0 && f.first == lines[f.at].Text {
		f.words = lines[f.at].Words
	}
	return f
}

// floatText is the line of synced lyrics sung at pos, with the line in
// Latin letters under it when the user shows them so, or else the next
// line; before the first line, a note over it.
func floatText(lines []lyrics.Line, latin []string, how string, pos time.Duration) floatLyric {
	at := sort.Search(len(lines), func(i int) bool { return ms(lines[i].TimeMs) > pos }) - 1
	if at < 0 {
		return floatLyric{at: -1, first: "♪", second: notchLine(lines, latin, how, 0)}
	}
	f := floatLyric{at: at, first: notchLine(lines, latin, how, at), second: notchUnder(latin, how, at)}
	if f.second == "" && at+1 < len(lines) {
		f.second = notchLine(lines, latin, how, at+1)
	}
	return f
}
