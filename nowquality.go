package main

// The now playing page tells, under the credits, what the track playing's
// audio is, as "FLAC · 24-bit · 96 kHz · 2304 kbit/s", from its Track
// info; clicked, it opens the Track info.

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/locales"
)

// nowQualityView is the track the line tells of, when it started telling,
// and when it last asked.
type nowQualityView struct {
	id           string
	since, asked time.Time
}

// nowQualityLine shows the quality of t, the track playing, once known.
func (s *shell) nowQualityLine(c *ui.Context, t media.Track) {
	k := &s.lk
	v := &s.nowQuality
	now := s.playingSource(t)
	key := "now-quality:" + t.ID
	l := fetch(s, key, func(ctx context.Context) (app.TrackInfo, error) { return trackInfoOf(s, ctx, t, now) })
	at := c.Now()
	if v.id != t.ID {
		*v = nowQualityView{id: t.ID, since: at, asked: at}
	}
	// As the dialog: the engine opens the track a moment after it's asked.
	if l.done && l.err == nil && (l.value.Playing != (now != nil) || waitsForEngine(l.value, now)) &&
		at.Sub(v.asked) >= time.Second && at.Sub(v.since) < trackInfoWait {
		v.asked = at
		s.refetch(key)
	}
	if !l.done || l.err != nil {
		return
	}
	line := qualityLine(l.value)
	if line == "" {
		return
	}
	b := s.buttonBase(c.Key("now-quality")).Label(locales.TrackInfoTitle()).Cursor(ui.CursorPointer)
	if b.Clicked() {
		s.showTrackInfo(t)
	}
	b.Children(func() {
		ui.Text(c, line).FontSize(k.smallSize()).TextColor(k.muted)
	})
}

// qualityLine is info's codec, bit depth, sample rate and bitrate, those
// it knows, joined by " · ": "" for a track Spotify plays, whose audio
// MixMusic doesn't see.
func qualityLine(info app.TrackInfo) string {
	if info.Plays == app.PlaysSpotify || info.Plays == app.PlaysConnect {
		return ""
	}
	f, ok := formatOf(info)
	st := info.Stream
	var parts []string
	switch {
	case ok && f.Codec != "":
		parts = append(parts, f.Codec)
	case st != nil && st.Codec != "":
		parts = append(parts, st.Codec)
	}
	if ok && f.BitDepth > 0 {
		parts = append(parts, locales.TrackInfoBitDepth(f.BitDepth))
	}
	if ok && f.SampleRate > 0 {
		parts = append(parts, kilohertz(f.SampleRate))
	}
	switch file := info.File; {
	case st != nil && st.Bitrate > 0:
		parts = append(parts, locales.TrackInfoBitrateValue(int(st.Bitrate+0.5)))
	case file != nil && file.Bitrate() > 0:
		parts = append(parts, locales.TrackInfoBitrateValue(int(file.Bitrate()+0.5)))
	}
	return strings.Join(parts, " · ")
}
