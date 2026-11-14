package main

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/player"
)

// TestVisualizerOfSilence draws nothing while MixMusic hears nothing, as
// of a Spotify device elsewhere: no flat line of bars, which would look
// broken.
func TestVisualizerOfSilence(t *testing.T) {
	s := newTestShell(t)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(240, 240).Background(ui.RGB(0, 0, 0)).Children(func() { s.visualizer(c, 240) })
	}, 240, 240)
	tt.Frame()
	img := tt.Image()
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			if r, g, b, _ := img.At(x, y).RGBA(); r|g|b != 0 {
				t.Fatalf("silence drew at (%d, %d)", x, y)
			}
		}
	}
}

// TestNowPlayingFillsTheWindow checks that the now playing page's lyrics
// take the window's height beside the sleeve: a taller window gives them
// as much more room.
func TestNowPlayingFillsTheWindow(t *testing.T) {
	s := newTestShell(t)
	s.p.PlayTracks(testTracks, 0)
	s.settle(t, func(st player.State) bool { return st.Playing })
	s.navigate("/now-playing")
	s.panel = ""
	tt := ui.NewTester(s.view, 1280, 820)
	for range 4 {
		tt.Frame()
	}
	short := s.lyricsView.room
	tt.SetSize(1280, 1220)
	for range 4 {
		tt.Frame()
	}
	if tall := s.lyricsView.room; short < 420 || tall-short < 399 || tall-short > 401 {
		t.Errorf("lyrics took %v in a window 820 high, %v in one 1220 high", short, tall)
	}
}

// TestNowPlayingShortWindow keeps the lyrics room to the window's real
// height, however short: held at 300, the sleeve and its credits
// overflowed a page that doesn't scroll.
func TestNowPlayingShortWindow(t *testing.T) {
	s := newTestShell(t)
	s.p.PlayTracks(testTracks, 0)
	s.settle(t, func(st player.State) bool { return st.Playing })
	s.navigate("/now-playing")
	s.panel = ""
	tt := ui.NewTester(s.view, 1280, 480)
	for range 4 {
		tt.Frame()
	}
	if room := s.lyricsView.room; room <= 0 || room >= 300 {
		t.Errorf("lyrics took %v in a window 480 high", room)
	}
}
