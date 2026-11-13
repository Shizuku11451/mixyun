package app

// Status bar lyrics: the line of the lyrics sung now, beside the tray icon
// in macOS's menu bar, or in Options.TaskbarLyrics's window over Windows'
// taskbar. They show while the setting is on, the main window is open, in
// the tray too, and a track with timed lyrics plays.

import (
	"context"
	"runtime"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"

	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/internal/player"
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
	// shown is the line shown, "" for none.
	shown string
	// asked is set while placeStatusLyrics waits for the main thread.
	asked bool
	// win is the window over Windows' taskbar, while open. Main thread.
	win *mygo.Window
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
	return a.status.shown
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
	text := ""
	if p := a.goPlayer.Load(); p != nil && main && a.settings.Get().StatusBarLyrics {
		st := p.State()
		if cur, ok := st.Current(); ok && !cur.Track.IsEpisode() {
			text = a.statusLine(cur.Track, st)
		}
	}
	s.mu.Lock()
	changed := text != s.shown
	s.shown = text
	s.mu.Unlock()
	if !changed {
		return
	}
	if runtime.GOOS == "darwin" {
		if tray != nil {
			tray.SetTitle(clip(text, menuBarRunes))
		}
		return
	}
	a.showTaskbarLyrics(text != "")
}

// statusLine is the line of t's lyrics sung at st's position, and asks
// for its lyrics once it is a new track.
func (a *App) statusLine(t media.Track, st player.State) string {
	s := &a.status
	s.mu.Lock()
	if s.track != t.ID {
		s.track, s.lines = t.ID, nil
		go a.loadStatusLyrics(t)
	}
	lines := s.lines
	s.mu.Unlock()
	return lineAt(lines, st.Position+statusLead(st.Speed))
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

// showTaskbarLyrics opens the window over Windows' taskbar, and redraws
// it, or closes it. Main thread.
func (a *App) showTaskbarLyrics(on bool) {
	s := &a.status
	switch {
	case on && s.win == nil && a.opts.TaskbarLyrics != nil:
		win := a.opts.TaskbarLyrics(a)
		if win == nil {
			return
		}
		s.win = win
		win.OnClosed(func() {
			if s.win == win {
				s.win = nil
			}
		})
	case !on && s.win != nil:
		s.win.Close()
		s.win = nil
	}
	if s.win != nil {
		s.win.Invalidate()
	}
}

// lineAt is the line of timed lyrics sung at pos: "♪" before the first,
// and between lines; "" without lyrics.
func lineAt(lines []lyrics.Line, pos time.Duration) string {
	if len(lines) == 0 {
		return ""
	}
	at := sort.Search(len(lines), func(i int) bool { return time.Duration(lines[i].TimeMs)*time.Millisecond > pos }) - 1
	if at < 0 || lines[at].Text == "" {
		return "♪"
	}
	return lines[at].Text
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
