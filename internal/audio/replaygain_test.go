package audio

import (
	"context"
	"math"
	"strings"
	"testing"
)

func TestReplayGainApplied(t *testing.T) {
	// Two tracks of one album: the same album gain, each its own track gain.
	a := ReplayGain{TrackGain: "-7.50 dB", TrackPeak: "0.99", AlbumGain: "-6.20 dB", AlbumPeak: "1.0"}
	b := ReplayGain{TrackGain: "-3.10 dB", TrackPeak: "0.80", AlbumGain: "-6.20 dB", AlbumPeak: "1.0"}
	for _, c := range []struct {
		name string
		rg   ReplayGain
		mode ReplayGainMode
		want Gain
	}{
		{"a by track", a, ReplayGainTrack, Gain{Mode: ReplayGainTrack, DB: -7.5, Tagged: -7.5}},
		{"b by track", b, ReplayGainTrack, Gain{Mode: ReplayGainTrack, DB: -3.1, Tagged: -3.1}},
		{"a by album", a, ReplayGainAlbum, Gain{Mode: ReplayGainAlbum, DB: -6.2, Tagged: -6.2}},
		{"b by album", b, ReplayGainAlbum, Gain{Mode: ReplayGainAlbum, DB: -6.2, Tagged: -6.2}},
		{"no unit, any case", ReplayGain{TrackGain: " -2 DB"}, ReplayGainTrack, Gain{Mode: ReplayGainTrack, DB: -2, Tagged: -2}},
		{"missing", ReplayGain{TrackGain: "-3 dB"}, ReplayGainAlbum, Gain{Mode: ReplayGainAlbum, Problem: GainMissing}},
		{"malformed", ReplayGain{TrackGain: "loud"}, ReplayGainTrack, Gain{Mode: ReplayGainTrack, Problem: GainInvalid}},
		{"out of range", ReplayGain{TrackGain: "NaN dB"}, ReplayGainTrack, Gain{Mode: ReplayGainTrack, Problem: GainInvalid}},
		// A boost stops where the peak reaches full scale.
		{"peak holds a boost", ReplayGain{TrackGain: "+9 dB", TrackPeak: "0.5"}, ReplayGainTrack,
			Gain{Mode: ReplayGainTrack, DB: -20 * math.Log10(0.5), Tagged: 9, Limited: true}},
		{"boost under the peak", ReplayGain{TrackGain: "+3 dB", TrackPeak: "0.5"}, ReplayGainTrack,
			Gain{Mode: ReplayGainTrack, DB: 3, Tagged: 3}},
		{"no peak, no boost", ReplayGain{TrackGain: "+3 dB"}, ReplayGainTrack,
			Gain{Mode: ReplayGainTrack, Tagged: 3, Limited: true, NoPeak: true}},
		{"peak past full scale", ReplayGain{TrackGain: "+3 dB", TrackPeak: "1.2"}, ReplayGainTrack,
			Gain{Mode: ReplayGainTrack, Tagged: 3, Limited: true}},
	} {
		got, ok := c.rg.Applied(c.mode)
		if !ok || got != c.want {
			t.Errorf("%s: Applied = %+v, %v, want %+v", c.name, got, ok, c.want)
		}
	}
	for _, m := range []ReplayGainMode{"", ReplayGainOff} {
		if _, ok := a.Applied(m); ok {
			t.Errorf("mode %q applies", m)
		}
	}
}

// gainedSource is a memSource of a file with ReplayGain tags.
type gainedSource struct {
	memSource
	rg ReplayGain
}

func (g gainedSource) ReplayGain() ReplayGain { return g.rg }

// gainedOpener opens files as fileOpener does, with the track gain a URL
// tells after #, as "track.webm#-6 dB".
func gainedOpener(ctx context.Context, url string, interrupted func() bool) (Source, error) {
	path, gain, _ := strings.Cut(url, "#")
	src, err := fileOpener(ctx, path, interrupted)
	if err != nil {
		return nil, err
	}
	return gainedSource{src.(memSource), ReplayGain{TrackGain: gain, AlbumGain: "-1 dB"}}, nil
}

func TestEngineReplayGain(t *testing.T) {
	want := reference(t, cuesFront)
	for _, c := range []struct {
		url  string
		mode ReplayGainMode
		db   float64
	}{
		{cuesFront + "#-6 dB", ReplayGainTrack, -6},
		{cuesFront + "#-6 dB", ReplayGainAlbum, -1},
		{cuesFront + "#-6 dB", ReplayGainOff, 0},
		{cuesFront + "#bad", ReplayGainTrack, 0},
	} {
		dev := &fakeDevice{}
		e := NewEngine(gainedOpener, dev.start, newRecorder())
		dev.e = e
		e.master = newSmoother(1, glide)
		e.SetReplayGain(c.mode)
		e.Load(c.url, Cue{})
		buffered(t, e, 0, ringFrames/2)
		e.Play()
		got := dev.play(t, len(want)/2)
		g := float32(math.Pow(10, c.db/20))
		for i := 2 * fadeFrames; i < len(want); i++ {
			if w := want[i] * g; got[i] != w {
				t.Fatalf("%s, %s: frame %d: %v, want %v", c.url, c.mode, i/2, got[i], w)
			}
		}
		if l, _ := e.Loaded(); (l.ReplayGain != nil) != (c.mode != ReplayGainOff) {
			t.Errorf("%s, %s: Loaded.ReplayGain = %+v", c.url, c.mode, l.ReplayGain)
		}
		e.Close()
	}
}
