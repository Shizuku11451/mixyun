package app

import (
	"testing"
	"time"

	"github.com/fzn0x/mixmusic/lyrics"
)

// TestLinesAt shows the line sung at a position and the next with words, a
// note between lines, and nothing without lyrics.
func TestLinesAt(t *testing.T) {
	lines := []lyrics.Line{{TimeMs: 1000, Text: "one"}, {TimeMs: 3000, Text: ""}, {TimeMs: 5000, Text: "two"}}
	for pos, want := range map[time.Duration][2]string{
		0: {"♪", "one"}, time.Second: {"one", "two"}, 4 * time.Second: {"♪", "two"}, time.Minute: {"two", ""},
	} {
		if cur, next := linesAt(lines, pos); cur != want[0] || next != want[1] {
			t.Errorf("at %v: %q %q, not %q", pos, cur, next, want)
		}
	}
	if cur, next := linesAt(nil, time.Second); cur != "" || next != "" {
		t.Errorf("without lyrics: %q %q", cur, next)
	}
	if got := clip("ピョンピョンまた", 4); got != "ピョン…" {
		t.Errorf("clip: %q", got)
	}
}
