package localfiles

import (
	"errors"
	"testing"
	"time"
)

const sheet = `REM GENRE Jazz
PERFORMER "Ada Field"
TITLE "Kites"
FILE "Kites.flac" WAVE
  TRACK 01 AUDIO
    TITLE "Paper"
    INDEX 01 00:00:00
  TRACK 02 AUDIO
    TITLE "Wind"
    PERFORMER "Ada Field & Co"
    INDEX 00 01:59:00
    INDEX 01 02:00:30
  TRACK 03 AUDIO
    INDEX 01 04:10:74
`

// TestReadCue reads an album of one FLAC file: each track its part, from
// its INDEX 01 to the next one's, named as the sheet says.
func TestReadCue(t *testing.T) {
	ti := newIndex(t)
	flac := ti.write(t, "Kites/Kites.flac", tagged(t, "Whole", "Someone", "", 300), time.Hour)
	cue := ti.write(t, "Kites/Kites.cue", sheet, time.Hour)
	tracks, err := ti.ReadCue(cue)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		title, artist string
		start, end    int64
		ms            int
	}{
		{"Paper", "Ada Field", 0, 120*75 + 30, 120400},
		{"Wind", "Ada Field & Co", 120*75 + 30, 250*75 + 74, 130586},
		{"Track 03", "Ada Field", 250*75 + 74, 0, 49013},
	}
	if len(tracks) != len(want) {
		t.Fatalf("%d tracks", len(tracks))
	}
	for i, w := range want {
		tr := tracks[i]
		if tr.ID != CueTrackID(flac, w.start, w.end) || tr.Title != w.title || tr.Artists[0].Name != w.artist || tr.DurationMs != w.ms {
			t.Errorf("track %d: %s %q by %v, %d ms", i+1, tr.ID, tr.Title, tr.Artists, tr.DurationMs)
		}
		path, start, end, ok := CuePartOf(tr.ID)
		if !ok || path != flac || start != cdTime(w.start) || end != cdTime(w.end) {
			t.Errorf("track %d is %s from %v to %v", i+1, path, start, end)
		}
		if _, ok := PathOf(tr.ID); ok {
			t.Errorf("track %d's ID reads as a whole file's", i+1)
		}
	}
	if tracks[0].Album == nil || tracks[0].Album.Name != "Kites" {
		t.Errorf("album %v", tracks[0].Album)
	}
}

// TestReadCueFails names the line of a sheet that can't play as it says.
func TestReadCueFails(t *testing.T) {
	ti := newIndex(t)
	ti.write(t, "Kites/Kites.flac", tagged(t, "Whole", "Someone", "", 300), time.Hour)
	for _, tc := range []struct {
		sheet string
		line  int
		why   CueProblem
	}{
		{"FILE \"Kites.flac\" WAVE\nTRACK 01 AUDIO\nINDEX 01 00:00:00\nFILE \"More.flac\" WAVE", 4, CueUnsupported},
		{"FILE \"../Kites.flac\" WAVE", 1, CueBadFile},
		{"FILE \"/music/Kites.flac\" WAVE", 1, CueBadFile},
		{"FILE \"Kites.wav\" WAVE", 1, CueBadFile},
		{"FILE \"Gone.flac\" WAVE\nTRACK 01 AUDIO\nINDEX 01 00:00:00", 0, CueMissing},
		{"FILE \"Kites.flac\" WAVE\nTRACK 01 MODE1/2352\nINDEX 01 00:00:00", 2, CueUnsupported},
		{"FILE \"Kites.flac\" WAVE\nTRACK 01 AUDIO\nINDEX 01 00:61:00", 3, CueBadIndex},
		{"FILE \"Kites.flac\" WAVE\nTRACK 01 AUDIO\nINDEX 01 01:00:00\nTRACK 02 AUDIO\nINDEX 01 00:30:00", 5, CueBadIndex},
		{"FILE \"Kites.flac\" WAVE\nTRACK 01 AUDIO\nTITLE \"x\"", 2, CueBadIndex},
		{"FILE \"Kites.flac\" WAVE\nTRACK 01 AUDIO\nPREGAP 00:02:00\nINDEX 01 00:00:00", 3, CueUnsupported},
		{"FILE \"Kites.flac\" WAVE\nTRACK 01 AUDIO\nINDEX 01 00:00:00\nTRACK 02 AUDIO\nINDEX 01 05:00:00", 4, CueOutOfRange},
		{"REM nothing", 0, CueNoTracks},
	} {
		_, err := ti.ReadCue(ti.write(t, "Kites/Kites.cue", tc.sheet, time.Hour))
		var ce *CueError
		if !errors.As(err, &ce) || ce.Line != tc.line || ce.Problem != tc.why {
			t.Errorf("%q: %v, want line %d %s", tc.sheet, err, tc.line, tc.why)
		}
	}
}

func TestCuePartOfOthers(t *testing.T) {
	for _, id := range []string{TrackID("/music/a.flac"), "file:cue:x:1", "file:cue:L211c2lj:20:10", "yt:abc"} {
		if _, _, _, ok := CuePartOf(id); ok {
			t.Errorf("%s reads as a part of a file", id)
		}
	}
}
