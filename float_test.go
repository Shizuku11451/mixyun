package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/fzn0x/mixmusic/internal/player"

	"github.com/fzn0x/mixmusic/internal/settings"
	"github.com/fzn0x/mixmusic/lyrics"
)

func TestFloatText(t *testing.T) {
	lines := []lyrics.Line{{TimeMs: 1000, Text: "one"}, {TimeMs: 2000, Text: "two"}}
	for _, tc := range []struct {
		name          string
		latin         []string
		how           string
		pos           time.Duration
		first, second string
	}{
		{"before the first line", nil, "", 0, "♪", "one"},
		{"a line and the next", nil, "", 1500 * time.Millisecond, "one", "two"},
		{"the last line", nil, "", 3 * time.Second, "two", ""},
		{"Latin letters under it", []string{"wan", "tsu"}, settings.RomanizeUnder, 1500 * time.Millisecond, "one", "wan"},
	} {
		f := floatText(lines, tc.latin, tc.how, tc.pos)
		if f.first != tc.first || f.second != tc.second {
			t.Errorf("%s: %q, %q; want %q, %q", tc.name, f.first, f.second, tc.first, tc.second)
		}
	}
}

func TestFloatLayout(t *testing.T) {
	if l := floatLayoutOf(460, 128); l.art == 0 || !l.second || !l.bar || !l.meta {
		t.Errorf("at its first size: %+v; want the cover, two lines, the song and the transport", l)
	}
	if l := floatLayoutOf(260, 64); l.art != 0 || l.second || l.bar {
		t.Errorf("at its least: %+v; want the line sung alone", l)
	}
	if l := floatLayoutOf(800, 400); l.art > floatArtMax || l.size > 34 || l.lines != 2 {
		t.Errorf("large: %+v; want the cover and the words no larger than their most", l)
	}
}

func TestFloatControlsOnHover(t *testing.T) {
	s := newTestShell(t)
	s.p.PlayTracks(testTracks[:1], 0)
	s.settle(t, func(st player.State) bool { return st.Playing })
	tt := ui.NewTester(s.floatView, 460, 128)
	tt.Frame()
	seek := func() bool { _, ok := tt.Find("Seek"); return ok }
	if seek() {
		t.Error("the progress shows before the pointer comes")
	}
	tt.Move(230, 64)
	tt.Advance(time.Second)
	if !seek() {
		t.Fatal("no progress on hover")
	}
	tt.Move(-10, -10)
	tt.Advance(time.Second)
	if !seek() {
		t.Error("the progress went before floatHideAfter")
	}
	tt.Advance(2 * time.Second)
	if seek() {
		t.Error("the progress stays once the pointer left")
	}
}
