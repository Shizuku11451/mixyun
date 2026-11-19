package main

import (
	"testing"

	"github.com/fzn0x/mixmusic/internal/player"
	"github.com/fzn0x/mixmusic/internal/settings"
	"github.com/fzn0x/mixmusic/locales"
)

// TestVolumeKeepsItsWidth narrows the window to its least: the volume
// slider keeps the width it has in a wide one, and the controls beside it
// stay clear of the track's progress, with the bar crowded too: floating,
// with the Spotify devices and a speed of its own beside the volume.
func TestVolumeKeepsItsWidth(t *testing.T) {
	for _, crowded := range []bool{false, true} {
		s := newTestShell(t)
		if crowded {
			s.set.Layout = settings.LayoutExpressive
			wear(s, "material", nil)
			s.set.SpotifyPlayback = settings.SpotifyPlaybackConnect
			s.set.SongSpeed = 1.25
		}
		s.p.PlayTracks(testTracks, 0)
		s.settle(t, func(st player.State) bool { return st.Playing })
		var wide float32
		for _, w := range []int{1280, 940} {
			tt := testerOf(t, s, w, 820)
			vol, ok := tt.Find(locales.PlayerVolume())
			if !ok {
				t.Fatalf("crowded %v, at %d: no volume slider: %q", crowded, w, tt.Texts())
			}
			seek, _ := tt.Find(locales.PlayerSeek())
			first, _ := tt.Find(locales.PlayerStatusBarLyrics())
			if r, ok := tt.Find(s.lyricsTitle()); ok && (first.W == 0 || r.X < first.X) {
				first = r
			}
			if wide == 0 {
				wide = vol.W
			} else if vol.W != wide {
				t.Errorf("crowded %v, at %d: the volume slider is %v wide; want %v, as in a wide window", crowded, w, vol.W, wide)
			}
			if first.X < seek.X+seek.W {
				t.Errorf("crowded %v, at %d: the controls start at %v, over the progress, which ends at %v", crowded, w, first.X, seek.X+seek.W)
			}
			if vol.X+vol.W > float32(w) {
				t.Errorf("crowded %v, at %d: the volume slider ends at %v, past the window", crowded, w, vol.X+vol.W)
			}
		}
	}
}
