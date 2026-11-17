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

func TestFloatScrollBy(t *testing.T) {
	for _, tc := range []struct{ x, want float32 }{{0, 0}, {40, 0}, {100, 60}, {600, 400}} {
		if got := floatScrollBy(tc.x, 500, 100); got != tc.want {
			t.Errorf("sung %v into a line 500 wide in 100: scrolled %v; want %v", tc.x, got, tc.want)
		}
	}
}

func TestSungX(t *testing.T) {
	words := []lyrics.Word{{Text: "one", StartMs: 0, EndMs: 1000}, {Text: "two", StartMs: 2000, EndMs: 3000}}
	g := shapeSung(sungKey{text: "one two", font: ui.Font{Size: 20}, width: 1000}, words)
	one, two := g.rows[0].cells[0], g.rows[0].cells[1]
	for _, tc := range []struct {
		name string
		at   time.Duration
		want float32
	}{
		{"before", -time.Second, 0},
		{"half the first word", 500 * time.Millisecond, (one.x0 + one.x1) / 2},
		{"between the words", 1500 * time.Millisecond, one.x1},
		{"half the second word", 2500 * time.Millisecond, (two.x0 + two.x1) / 2},
		{"after", 5 * time.Second, two.x1},
	} {
		if got := g.sungX(words, tc.at); got < tc.want-0.01 || got > tc.want+0.01 {
			t.Errorf("%s: %v; want %v", tc.name, got, tc.want)
		}
	}
}

func TestFloatLayout(t *testing.T) {
	if l := floatLayoutOf(460, 128); l.art == 0 || !l.second || !l.bar || !l.meta {
		t.Errorf("at its first size: %+v; want the cover, the line under, the song and the transport", l)
	}
	if l := floatLayoutOf(260, 64); l.art != 0 || l.second || l.bar {
		t.Errorf("at its least: %+v; want the line sung alone", l)
	}
	if l := floatLayoutOf(800, 400); l.art > floatArtMax || l.size > 34 {
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

func TestFloatLocked(t *testing.T) {
	s := newTestShell(t)
	s.p.PlayTracks(testTracks[:1], 0)
	s.settle(t, func(st player.State) bool { return st.Playing })
	s.lockFloat(true)
	tt := ui.NewTester(s.floatView, 460, 128)
	tt.Frame()
	if _, ok := tt.Find("Unlock the floating window"); ok {
		t.Error("the unlock button shows before the pointer comes")
	}
	s.fl.hover = true // as pollFloatPointer finds it, and redraws
	tt.Frame()
	tt.Advance(time.Second)
	if _, ok := tt.Find("Seek"); ok {
		t.Error("the progress shows on a locked window")
	}
	if err := tt.Click("Unlock the floating window"); err != nil {
		t.Fatalf("no unlock button over a locked window: %v; it shows %q", err, tt.Texts())
	}
	tt.Frame()
	if s.set.FloatLocked {
		t.Error("still locked once unlocked")
	}
}

// TestFloatLockedLegible keeps the words of a locked window, lit word by
// word, white over their dark edge: the line paints its glyphs in Draw,
// under its own text's edge and shadow, which then hid them.
func TestFloatLockedLegible(t *testing.T) {
	s := newTestShell(t)
	s.loads["lyrics:"+testTracks[0].ID] = &loaded[lyrics.Lyrics]{done: true, value: lyrics.Lyrics{Synced: true, Source: "Unison", Lines: lanterns}}
	s.p.PlayTracks(testTracks, 0)
	s.settle(t, func(st player.State) bool { return st.Playing && !st.Loading })
	s.p.Time(3500*time.Millisecond, 200*time.Second)
	s.settle(t, func(st player.State) bool { return st.Position == 3500*time.Millisecond })
	s.lockFloat(true)
	tt := ui.NewTester(s.floatView, 460, 128)
	tt.Frame()
	tt.Advance(time.Second)
	img := tt.Image()
	white := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if c := img.RGBAAt(x, y); c.A > 240 && c.R > 240 && c.G > 240 && c.B > 240 {
				white++
			}
		}
	}
	if white < 300 {
		t.Errorf("%d white pixels: the words are covered", white)
	}
}

// TestFloatLineGrows grows the next line out of where it showed under the
// line sung, up to that line's place, as the line sung goes.
func TestFloatLineGrows(t *testing.T) {
	s := newTestShell(t)
	s.loads["lyrics:"+testTracks[0].ID] = &loaded[lyrics.Lyrics]{done: true, value: lyrics.Lyrics{Synced: true, Source: "Unison", Lines: lanternLines}}
	s.p.PlayTracks(testTracks, 0)
	s.settle(t, func(st player.State) bool { return st.Playing && !st.Loading })
	s.p.Time(3500*time.Millisecond, 200*time.Second)
	s.settle(t, func(st player.State) bool { return st.Position == 3500*time.Millisecond })
	tt := ui.NewTester(s.floatView, 460, 128)
	tt.Frame()
	tt.Advance(time.Second)
	sung, ok := tt.Find("Lanterns drifting home")
	under, ok2 := tt.Find("Quiet harbour")
	if !ok || !ok2 || under.Y <= sung.Y {
		t.Fatalf("the lines at %v and %v; it shows %q", sung, under, tt.Texts())
	}
	s.p.Time(60500*time.Millisecond, 200*time.Second)
	s.settle(t, func(st player.State) bool { return st.Position == 60500*time.Millisecond })
	tt.Frame()
	start, _ := tt.Find("Quiet harbour")
	tt.Advance(time.Second)
	end, ok := tt.Find("Quiet harbour")
	if !ok || start.Y < end.Y+4 || end.H < under.H*1.3 || start.H > end.H*0.8 {
		t.Errorf("the next line from %v to %v; it was %v under %v", start, end, under, sung)
	}
	if tt.HasText("Lanterns drifting home") {
		t.Error("the line sung before stays")
	}
}
