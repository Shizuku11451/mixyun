package main

import (
	"testing"
	"time"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/internal/audio"
)

// TestQualityLine tells a file's and a stream's audio, and nothing of a
// track Spotify plays.
func TestQualityLine(t *testing.T) {
	file := &app.TrackFile{Kind: app.FileMusic, Size: 23_000_000,
		Format: audio.Format{Codec: "FLAC", SampleRate: 96000, BitDepth: 24, Duration: 100 * time.Second}}
	for _, c := range []struct {
		info app.TrackInfo
		want string
	}{
		{app.TrackInfo{Plays: app.PlaysEngine, File: file}, "FLAC · 24-bit · 96 kHz · 1840 kbit/s"},
		{app.TrackInfo{Plays: app.PlaysEngine, Stream: &app.TrackStream{Codec: "Opus", Bitrate: 139.3}}, "Opus · 139 kbit/s"},
		{app.TrackInfo{Plays: app.PlaysSpotify}, ""},
	} {
		if got := qualityLine(c.info); got != c.want {
			t.Errorf("qualityLine = %q, want %q", got, c.want)
		}
	}
}
