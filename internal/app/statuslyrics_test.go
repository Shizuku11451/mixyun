package app

import (
	"testing"
	"time"

	"github.com/fzn0x/mixmusic/internal/settings"
	"github.com/fzn0x/mixmusic/lyrics"
)

// TestLinesAt shows the line sung at a position and the next with words, a
// note between lines, and nothing without lyrics.
func TestLinesAt(t *testing.T) {
	lines := []lyrics.Line{{TimeMs: 1000, Text: "one"}, {TimeMs: 3000, Text: ""}, {TimeMs: 5000, Text: "two"}}
	for pos, want := range map[time.Duration][2]string{
		0: {"♪", "one"}, time.Second: {"one", "two"}, 4 * time.Second: {"♪", "two"}, time.Minute: {"two", ""},
	} {
		if cur, next, _ := linesAt(lines, pos); cur != want[0] || next != want[1] {
			t.Errorf("at %v: %q %q, not %q", pos, cur, next, want)
		}
	}
	if cur, next, _ := linesAt(nil, time.Second); cur != "" || next != "" {
		t.Errorf("without lyrics: %q %q", cur, next)
	}
	timed := []lyrics.Line{{TimeMs: 1000, Text: "a b", Words: []lyrics.Word{{Text: "a"}, {Text: "b"}}}}
	if _, _, words := linesAt(timed, 2*time.Second); len(words) != 2 {
		t.Errorf("words: %v", words)
	}
	if got := clip("ピョンピョンまた", 4); got != "ピョン…" {
		t.Errorf("clip: %q", got)
	}
}

// TestStatusTranslateTo translates the taskbar's lyrics as the lyrics are:
// for a language the user translates, into the one chosen, and not into
// their own.
func TestStatusTranslateTo(t *testing.T) {
	a := newQuietApp(t, nil)
	a.settings.Update(func(v *settings.Settings) error {
		v.LyricsTranslate, v.LyricsTranslateTo = []string{"ja"}, "zh-Hans"
		return nil
	})
	for lang, want := range map[string]string{"ja": "zh-Hans", "en": "", "": ""} {
		if got := a.statusTranslateTo(lyrics.Lyrics{Language: lang}); got != want {
			t.Errorf("lyrics in %q translate into %q, want %q", lang, got, want)
		}
	}
	a.settings.Update(func(v *settings.Settings) error { v.LyricsTranslateTo = "ja"; return nil })
	if got := a.statusTranslateTo(lyrics.Lyrics{Language: "ja"}); got != "" {
		t.Errorf("lyrics in Japanese translate into their own language: %q", got)
	}
}
