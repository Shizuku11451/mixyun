package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
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
const floatArtMax = 320

// floatPollEvery is how often a locked floating window, which lets the
// pointer through, looks where the pointer is (floatLock).
const floatPollEvery = 100 * time.Millisecond

// floatLock is the floating window's lock: whether the window is locked as
// the setting FloatLocked says (applied), whether the pointer is over it
// (hover) and over its unlock button (overButton), where that button was
// the frame before, and stop, which ends the polling of the pointer.
type floatLock struct {
	applied, hover, overButton bool
	button                     ui.Rect
	stop                       func()
}

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
		Vibrancy: mygo.VibrancyGlass, // as unlocking puts it back (followFloatLock)
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
	s.follow(func() { s.pollFloatPointer(false) })
	return s.win
}

// floatLayout is what the floating lyrics show at a window's size: the
// cover where it is wide enough, the line under the one sung where it is
// tall enough, and the song and the transport under them where it is
// taller still.
type floatLayout struct {
	pad, art, size           float32
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
	if a := min(inner, w*0.28, floatArtMax, room); a >= 40 {
		l.art = a
	}
	column := w - 2*l.pad - l.art
	if l.art > 0 {
		column -= l.pad
	}
	// The words grow with the room the window gives them, so that a large
	// window is not mostly empty: about a third of its height with the line
	// under, half without, and narrow enough to show a few words across.
	share := float32(0.3)
	if !l.second {
		share = 0.5
	}
	l.size = max(15, min(inner*share, column*0.09, 64))
	l.meta = h >= 100
	l.times = column >= 290
	return l
}

// floatView draws the floating lyrics. Dragging it anywhere but its
// buttons moves the window, and its edges resize it; its menu, on a right
// click, plays or pauses, opens MixMusic, locks it or closes it. Locked
// (followFloatLock), it shows the lyrics alone, with no ground, and its
// unlock button while the pointer is over it.
func (s *shell) floatView(c *ui.Context) {
	s.st = s.p.State()
	s.followArtwork()
	k := &s.lk
	t := k.theme
	t.Background = ui.Transparent
	c.SetTheme(&t)
	l := floatLayoutOf(c.Size())
	locked := s.followFloatLock()
	if locked {
		l.art, l.bar, l.meta = 0, false, false
	}
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
	if locked {
		ground = ui.Transparent
	}
	root := ui.Row(c).Fill().Background(ground).Radius(22).Padding(l.pad).Gap(l.pad).AlignItems(ui.Center).
		TextColor(k.text).Font(k.body).DragWindow()
	if locked {
		// A dark edge round the white words (floatLyricView), crisp on light
		// ground as on dark; a blurred shadow alone reads as a smudge on white.
		root.TextStroke(3, ui.RGBA(0, 0, 0, 0.6)).TextShadow(0, 1, 2, ui.RGBA(0, 0, 0, 0.35))
	}
	root.ContextMenu(func(m *ui.Menu) {
		// The menu plays and pauses too: the transport shows only on hover.
		if _, label := toggleOf(s.st); ok && m.Item(label).Chosen() {
			s.p.Toggle()
		}
		if m.Item(locales.MiniRowOpen()).Chosen() {
			go s.svc.System.ShowMainWindow()
		}
		if m.Item(locales.FloatLock()).Chosen() {
			s.lockFloat(true)
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
	unlock := float32(0)
	if locked {
		// It stays as the progress does once the pointer leaves.
		on := s.fl.hover || s.fl.overButton
		a := ui.Smooth
		if !on {
			a.Delay = floatHideAfter
		}
		unlock = root.AnimateTo("unlock", b2f(on), a)
	}
	root.Children(func() {
		if unlock > 0.01 {
			b := s.miniButton(c, "float-unlock", "unlock", locales.FloatUnlock(), false, 28, 16).
				Absolute().Top(4).Right(4).Background(k.surface).Opacity(unlock)
			s.fl.button = b.Bounds()
			if b.Clicked() {
				s.lockFloat(false)
			}
		} else {
			s.fl.button = ui.Rect{}
		}
		if l.art > 0 {
			s.floatCover(c, cur.Track, ok, l.art)
		}
		ui.Column(c).Grow(1).AlignSelf(ui.Stretch).MinWidth(0).Children(func() {
			if l.meta {
				meta := max(11, l.size*0.38)
				ui.Row(c).Gap(6).MinWidth(0).Children(func() {
					ui.Text(c, title).FontSize(meta).FontWeight(600).SingleLine().Shrink(0).MaxWidth(meta * 16)
					ui.Text(c, artists).FontSize(meta).TextColor(k.muted).SingleLine().MinWidth(0)
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
// last rises out. The line keeps to one row: one longer than the window
// scrolls across it as it is sung (floatScrollBy).
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
		second := k.muted
		view := ui.Box(c).MinWidth(0).AlignItems(ui.Start).Clip()
		view.Children(func() {
			var line, sung ui.Element
			if !s.set.FloatLocked {
				line = s.lyricsText(c, ly.first, l.size, 700, k.accent, ly.words, 0, 0)
				sung = line
			} else {
				// No ground under them: the cover's colours could be the wallpaper's.
				white := ui.RGB(255, 255, 255)
				second = white.Alpha(0.85)
				text := func() ui.Element { return ui.Text(c, ly.first).FontSize(l.size).FontWeight(700).NoWrap() }
				if len(ly.words) == 0 {
					line = text().TextColor(white)
					sung = line
				} else {
					// A line lit word by word paints its glyphs in Draw, under its
					// own text's edge and shadow: those come from a clear copy
					// behind it, and it takes none.
					line = ui.Box(c).Children(func() {
						text().TextColor(ui.Transparent)
						sung = s.sungText(c, text(), ly.first, ui.Font{Family: k.body, Size: l.size, Weight: 700}, white, white.Alpha(0.55), ly.words, 0, 0).
							Absolute().Top(0).Left(0).TextStroke(0, ui.Transparent).TextShadow(0, 0, 0, ui.Transparent)
					})
				}
			}
			line.NoWrap().Shrink(0)
			// The widths are the frame before's: a new line starts unscrolled.
			lineW, viewW := line.Bounds().W, view.Bounds().W
			if lineW <= viewW || ly.to <= ly.from {
				return
			}
			clock := ui.Local(view, "clock", func() sungLine { return sungLine{} }).follow(s.st, c.Now())
			if clock.moving {
				view.Loop("scroll", time.Second, ui.Linear) // a frame each, while it scrolls
			}
			at := clock.at(c.Now()) + lyricsLead(s.st.Speed)
			// Where it is sung: by its words, as they light (sungTextWith
			// keeps their layout on the line), or else by its time.
			x := lineW * min(max(float32(at-ly.from)/float32(ly.to-ly.from), 0), 1)
			if g := ui.Local(sung, "sung", func() sungLine { return sungLine{} }).shaped; len(ly.words) > 0 && g != nil {
				x = g.sungX(ly.words, at)
			}
			line.Translate(-floatScrollBy(x, lineW, viewW), 0)
		})
		if l.second && ly.second != "" {
			ui.Text(c, ly.second).FontSize(max(11, l.size*0.62)).TextColor(second).SingleLine().MinWidth(0).Opacity(under)
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
		if s.miniButton(c, "float-lock", "lock", locales.FloatLock(), false, 30, 16).Clicked() {
			s.lockFloat(true)
		}
	})
}

// lockFloat locks the floating window, or unlocks it, in the settings,
// which every window follows: the floating one with followFloatLock, the
// tray's and the player bar's items with their checks.
func (s *shell) lockFloat(on bool) {
	s.set.FloatLocked = on
	s.save()
}

// followFloatLock locks the floating window, or unlocks it, once the
// setting changed, and reports whether it is locked. Locked, the window
// lets the pointer through, and shows no glass nor shadow; it looks where
// the pointer is every floatPollEvery to show its unlock button, and lets
// the pointer in over that button alone (pollFloatPointer).
func (s *shell) followFloatLock() bool {
	on := s.set.FloatLocked
	if on == s.fl.applied || s.win == nil {
		return on
	}
	s.fl.applied = on
	w := s.win
	w.SetHasShadow(!on)
	w.SetIgnoreMouseEvents(on)
	if !on {
		w.SetVibrancy(mygo.VibrancyGlass)
		s.pollFloatPointer(false)
		return on
	}
	w.SetVibrancy(mygo.VibrancyNone)
	t, done := time.NewTicker(floatPollEvery), make(chan struct{})
	go func() {
		defer t.Stop()
		for {
			select {
			case <-t.C:
				mygo.RunOnMain(func() { s.pollFloatPointer(true) })
			case <-done:
				return
			}
		}
	}()
	s.fl.stop = sync.OnceFunc(func() { close(done) })
	return on
}

// pollFloatPointer looks where the pointer is over the locked floating
// window: over the window, it shows the unlock button, and over that
// button, it lets the pointer in, to click it. Called with on false, it
// stops looking, as the window unlocks or closes. Main thread.
func (s *shell) pollFloatPointer(on bool) {
	if !on {
		if s.fl.stop != nil {
			s.fl.stop()
			s.fl.stop = nil
		}
		// A poll queued on the main thread before may still run: it finds
		// the window unlocked, or closed, and looks no more.
		s.fl.applied, s.fl.hover, s.fl.overButton = false, false, false
		return
	}
	if !s.fl.applied {
		return
	}
	x, y, in := cursorIn(s.win)
	if over := in && s.fl.button.Contains(x, y); over != s.fl.overButton {
		s.fl.overButton = over
		s.win.SetIgnoreMouseEvents(!over)
	}
	if in != s.fl.hover {
		s.fl.hover = in
		s.win.Invalidate()
	}
}

// floatLyric is what the floating lyrics show of the lyrics: the line
// sung now, its index (-1 before the first, -2 for a note in place of
// lyrics), its words when the lyrics time them, the line under it, and
// when the line is sung from and to.
type floatLyric struct {
	at            int
	first, second string
	words         []lyrics.Word
	from, to      time.Duration
}

// floatScrollBy is how far a line lineW wide scrolls in a window viewW
// wide once it is sung x into it: none until x is 40% into the window,
// then as far as keeps x there, and no further than the line's end.
func floatScrollBy(x, lineW, viewW float32) float32 {
	return min(max(x-0.4*viewW, 0), max(lineW-viewW, 0))
}

// sungX is how far into the first row of the line the words are sung at
// at: the end of the last word sung, or the part sung of the one being
// sung, as sungLine.light lights them.
func (g *sungLayout) sungX(words []lyrics.Word, at time.Duration) float32 {
	if len(g.rows) == 0 {
		return 0
	}
	var x float32
	for _, cell := range g.rows[0].cells {
		w := words[cell.word]
		start, end := ms(w.StartMs), ms(w.EndMs)
		if at < start {
			continue
		}
		f := float32(1)
		if at < end {
			f = float32(at-start) / float32(end-start)
		}
		x = max(x, cell.x0+(cell.x1-cell.x0)*f)
	}
	return x
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
	f.from, f.to = ms(lines[at].TimeMs), ms(lines[at].TimeMs)+5*time.Second
	if at+1 < len(lines) {
		f.to = ms(lines[at+1].TimeMs)
	} else if w := lines[at].Words; len(w) > 0 {
		f.to = ms(w[len(w)-1].EndMs)
	}
	return f
}
