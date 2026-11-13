package app

import (
	"testing"
	"time"

	"github.com/fzn0x/mixmusic/lyrics"
)

// TestLineAt shows the line sung at a position, a note between lines, and
// nothing without lyrics.
func TestLineAt(t *testing.T) {
	lines := []lyrics.Line{{TimeMs: 1000, Text: "one"}, {TimeMs: 3000, Text: ""}, {TimeMs: 5000, Text: "two"}}
	for pos, want := range map[time.Duration]string{0: "♪", time.Second: "one", 4 * time.Second: "♪", time.Minute: "two"} {
		if got := lineAt(lines, pos); got != want {
			t.Errorf("at %v: %q, not %q", pos, got, want)
		}
	}
	if got := lineAt(nil, time.Second); got != "" {
		t.Errorf("without lyrics: %q", got)
	}
	if got := clip("ピョンピョンまた", 4); got != "ピョン…" {
		t.Errorf("clip: %q", got)
	}
}
