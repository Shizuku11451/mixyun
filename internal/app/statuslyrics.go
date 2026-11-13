package app

// Status bar lyrics: the line of the lyrics sung now, beside the tray icon
// in macOS's menu bar, or in a window of internal/taskbar in Windows'
// taskbar. They show while the setting is on, the main window is open, in
// the tray too, and a track with timed lyrics plays.

import (
	"bytes"
	"context"
	"errors"
	"image"
	"log"
	"runtime"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"

	"github.com/fzn0x/mixmusic/internal/artwork"
	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/internal/player"
	"github.com/fzn0x/mixmusic/internal/taskbar"
	"github.com/fzn0x/mixmusic/lyrics"
)

// menuBarRunes is how much of a line the menu bar shows, which has little
// room beside the other apps' items, and the notch.
const menuBarRunes = 36

// statusLyrics is what the status bar shows of the lyrics.
type statusLyrics struct {
	once sync.Once
	mu   sync.Mutex
	// track is the track whose lyrics lines are, or are asked for; lines
	// are nil meanwhile, and for a track without timed lyrics.
	track string
	lines []lyrics.Line
	// cover is the track's cover, for the taskbar, once it loaded; nil
	// meanwhile, and for a track without one.
	cover image.Image
	// shown is what shows: Current is "" for nothing.
	shown taskbar.LyricsView
	// asked is set while placeStatusLyrics waits for the main thread.
	asked bool
	// bar is the window in Windows' taskbar, while open, and failed is
	// set once it couldn't open, which is told of once. Main thread.
	bar    *taskbar.Lyrics
	failed bool
}

// followStatusLyrics shows the lyrics in the status bar as the player and
// the settings change, from the main window's opening.
func (a *App) followStatusLyrics() {
	a.status.once.Do(func() {
		if p := a.goPlayer.Load(); p != nil {
			p.Subscribe(func(player.Event) { a.placeStatusLyrics() })
		}
		a.Watch(a.placeStatusLyrics)
	})
	a.placeStatusLyrics()
}

// placeStatusLyrics has the main thread show the line sung now. It runs
// from any goroutine, the player's too, and returns at once.
func (a *App) placeStatusLyrics() {
	s := &a.status
	s.mu.Lock()
	asked := s.asked
	s.asked = true
	s.mu.Unlock()
	if !asked {
		mygo.App.WhenReady(a.applyStatusLyrics)
	}
}

// StatusLyric is the line the status bar shows, "" for none.
func (a *App) StatusLyric() string {
	a.status.mu.Lock()
	defer a.status.mu.Unlock()
	return a.status.shown.Current
}

// applyStatusLyrics shows the line sung now, on the main thread.
func (a *App) applyStatusLyrics() {
	s := &a.status
	s.mu.Lock()
	s.asked = false
	s.mu.Unlock()
	a.mu.Lock()
	main, tray := a.main != nil, a.tray
	a.mu.Unlock()
	var v taskbar.LyricsView
	if p := a.goPlayer.Load(); p != nil && main && a.settings.Get().StatusBarLyrics {
		st := p.State()
		if cur, ok := st.Current(); ok && !cur.Track.IsEpisode() {
			v = a.statusView(cur.Track, st)
		}
	}
	s.mu.Lock()
	changed := v != s.shown
	s.shown = v
	s.mu.Unlock()
	if !changed {
		return
	}
	if runtime.GOOS == "darwin" {
		if tray != nil {
			tray.SetTitle(clip(v.Current, menuBarRunes))
		}
		return
	}
	a.showTaskbarLyrics(v)
}

// statusView is the lines of t's lyrics sung at st's position, and the one
// after, with t's cover, and asks for its lyrics and cover once it is a new
// track. Without lyrics, it shows nothing.
func (a *App) statusView(t media.Track, st player.State) taskbar.LyricsView {
	s := &a.status
	s.mu.Lock()
	if s.track != t.ID {
		s.track, s.lines, s.cover = t.ID, nil, nil
		go a.loadStatusLyrics(t)
		if runtime.GOOS == "windows" {
			go a.loadStatusCover(t)
		}
	}
	lines, cover := s.lines, s.cover
	s.mu.Unlock()
	cur, next := linesAt(lines, st.Position+statusLead(st.Speed))
	if cur == "" {
		return taskbar.LyricsView{}
	}
	return taskbar.LyricsView{Current: cur, Next: next, Cover: cover}
}

// statusCoverSize is how big, in pixels, the taskbar's cover is kept: the
// taskbar scales it down to its 34 DIPs, which is 68 pixels at 200 %.
const statusCoverSize = 96

// loadStatusCover loads the cover of t for the taskbar, square.
func (a *App) loadStatusCover(t media.Track) {
	u := controlsArtwork(t.Artwork)
	if u == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	data, err := a.coverBytes(ctx, u)
	if err != nil || len(data) == 0 {
		return // the store logs why
	}
	square, err := artwork.Square(data, statusCoverSize)
	if err != nil {
		return
	}
	img, _, err := image.Decode(bytes.NewReader(square))
	if err != nil {
		return
	}
	s := &a.status
	s.mu.Lock()
	if s.track == t.ID {
		s.cover = img
	}
	s.mu.Unlock()
	a.placeStatusLyrics()
}

// loadStatusLyrics asks for the lyrics of t, which Catalog.Lyrics keeps.
// shortcut: a lookup that fails isn't tried again until t plays again.
func (a *App) loadStatusLyrics(t media.Track) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	l, err := (&Catalog{a}).Lyrics(ctx, t)
	if err != nil || !l.Synced || l.Instrumental || len(l.Lines) == 0 {
		return
	}
	s := &a.status
	s.mu.Lock()
	if s.track == t.ID {
		s.lines = l.Lines
	}
	s.mu.Unlock()
	a.placeStatusLyrics()
}

// taskbarLyricsWidth is how wide the lyrics are in Windows' taskbar, in
// DIPs.
const taskbarLyricsWidth = 360

// showTaskbarLyrics shows v in Windows' taskbar, opening its window, or
// closes the window without v.Current. Main thread.
func (a *App) showTaskbarLyrics(v taskbar.LyricsView) {
	s := &a.status
	if s.bar != nil && (v.Current == "" || s.bar.Closed()) {
		// Gone with Explorer as it restarts, it opens again for this line.
		s.bar.Close()
		s.bar = nil
	}
	if v.Current == "" {
		return
	}
	if s.bar == nil {
		bar, err := taskbar.OpenLyrics(taskbarLyricsWidth)
		if err != nil {
			if !errors.Is(err, taskbar.ErrUnsupported) && !s.failed {
				log.Printf("mixmusic: the taskbar's lyrics: %v", err)
			}
			s.failed = true
			return
		}
		s.bar, s.failed = bar, false
	}
	s.bar.Show(v)
}

// linesAt is the line of timed lyrics sung at pos, and the next line with
// words: "♪" for the line before the first, and between lines; "" for both
// without lyrics, and for the next one after the last.
func linesAt(lines []lyrics.Line, pos time.Duration) (cur, next string) {
	if len(lines) == 0 {
		return "", ""
	}
	at := sort.Search(len(lines), func(i int) bool { return time.Duration(lines[i].TimeMs)*time.Millisecond > pos }) - 1
	cur = "♪"
	if at >= 0 && lines[at].Text != "" {
		cur = lines[at].Text
	}
	for _, l := range lines[at+1:] {
		if l.Text != "" {
			return cur, l.Text
		}
	}
	return cur, ""
}

// statusLead is how far ahead of the position a line shows, as the
// lyrics panel's lyricsLead: the position comes every 250 ms.
func statusLead(speed float64) time.Duration {
	if speed <= 0 {
		speed = 1
	}
	return time.Duration(float64(250*time.Millisecond) * speed)
}

// clip is text cut to n runes, with an ellipsis where it was cut.
func clip(text string, n int) string {
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	return string([]rune(text)[:n-1]) + "…"
}
