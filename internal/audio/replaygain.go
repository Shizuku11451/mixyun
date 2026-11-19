package audio

import (
	"math"
	"strconv"
	"strings"
)

// ReplayGain is a track's ReplayGain tags, as its file writes them, as
// "-6.20 dB" and "0.988": "" for a tag it hasn't.
type ReplayGain struct {
	TrackGain, TrackPeak, AlbumGain, AlbumPeak string
}

// Gained is a Source whose file may tell its ReplayGain: a file on this
// computer. Streams aren't, and play as they did.
type Gained interface {
	ReplayGain() ReplayGain
}

// ReplayGainMode is which of a file's ReplayGain tags the engine plays it
// by: "" or "off" for none.
type ReplayGainMode string

const (
	ReplayGainOff   ReplayGainMode = "off"
	ReplayGainTrack ReplayGainMode = "track"
	ReplayGainAlbum ReplayGainMode = "album"
)

// on reports whether the mode plays by a tag.
func (m ReplayGainMode) on() bool { return m == ReplayGainTrack || m == ReplayGainAlbum }

// GainProblem is why a track's ReplayGain isn't applied.
type GainProblem string

const (
	// GainMissing is a file without the tag the mode asks for.
	GainMissing GainProblem = "missing"
	// GainInvalid is a file whose tag can't be read as a gain.
	GainInvalid GainProblem = "invalid"
)

// Gain is what ReplayGain makes of a track.
type Gain struct {
	Mode ReplayGainMode
	// DB is the gain the track plays at, in dB, and Tagged the one its tag
	// asks for: they differ when Limited, the peak holding back a boost.
	DB, Tagged float64
	Limited    bool
	// NoPeak is set when a boost is held back as no peak tag tells how far
	// the track may go.
	NoPeak bool
	// Problem tells why no gain applies, "" when one does.
	Problem GainProblem
}

// Applied returns what ReplayGain makes of g in mode, and false while mode
// is off. A gain that would turn the track up stops where its peak, as its
// tag tells, reaches full scale; without a peak, it doesn't turn it up.
func (g ReplayGain) Applied(mode ReplayGainMode) (Gain, bool) {
	if !mode.on() {
		return Gain{}, false
	}
	gain, peak := g.TrackGain, g.TrackPeak
	if mode == ReplayGainAlbum {
		gain, peak = g.AlbumGain, g.AlbumPeak
	}
	out := Gain{Mode: mode}
	if strings.TrimSpace(gain) == "" {
		out.Problem = GainMissing
		return out, true
	}
	db, ok := parseGain(gain)
	if !ok {
		out.Problem = GainInvalid
		return out, true
	}
	out.DB, out.Tagged = db, db
	if db > 0 {
		p, err := strconv.ParseFloat(strings.TrimSpace(peak), 64)
		switch {
		case err != nil || !(p > 0) || math.IsInf(p, 0):
			out.DB, out.Limited, out.NoPeak = 0, true, true
		case db > -20*math.Log10(p):
			out.DB, out.Limited = max(-20*math.Log10(p), 0), true
		}
	}
	return out, true
}

// parseGain reads a gain as ReplayGain writes it: "-6.20 dB", the unit
// left out or in any case. One past ±64 dB is no gain a scanner writes.
func parseGain(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && strings.EqualFold(s[len(s)-2:], "db") {
		s = strings.TrimSpace(s[:len(s)-2])
	}
	db, err := strconv.ParseFloat(s, 64)
	if err != nil || !(db >= -64 && db <= 64) {
		return 0, false
	}
	return db, true
}
