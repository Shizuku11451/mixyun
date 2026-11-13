package lyrics

import (
	"slices"
	"strings"
	"testing"
)

// TestParseYRC reads NetEase's lyrics synchronized word by word: the
// words with their times, each found in its line's text after the one
// before, and no line of the credits.
func TestParseYRC(t *testing.T) {
	lines := ParseYRC(`{"t":0,"c":[{"tx":"作词: "},{"tx":"someone"}]}
[11460,2730](11460,210,0)I (11670,180,0)do (11850,120,0)the (11970,540,0)same
[0,11460](0,10860,0)（(10860,60,0)The (10920,240,0)Kid(11460,0,0)）
[28480,900](28480,160,0)我(28640,420,0)带  (29060,320,0)着 
[40000,10]
`)
	if len(lines) != 3 || lines[0].Text != "（The Kid）" || lines[1].Text != "I do the same" || lines[2].Text != "我带  着" {
		t.Fatalf("lines: %+v", lines)
	}
	if w := lines[1].Words; len(w) != 4 || w[1] != (Word{Text: "do", StartMs: 11670, EndMs: 11850}) {
		t.Errorf("words: %+v", w)
	}
	for _, l := range lines {
		from := 0
		for _, w := range l.Words {
			at := strings.Index(l.Text[from:], w.Text)
			if at < 0 {
				t.Fatalf("%q isn't in %q after the word before", w.Text, l.Text)
			}
			from += at + len(w.Text)
		}
	}
}

// TestAlignLRC gives each line the translation sung at its time, or the
// nearest within half a second, and none to a line without one.
func TestAlignLRC(t *testing.T) {
	lines := []Line{{TimeMs: 11460}, {TimeMs: 14190}, {TimeMs: 20000}, {TimeMs: 30000}}
	got := AlignLRC(lines, "[by:someone]\n[00:11.310]one\n[00:13.959]two\n[00:19.400]far\n[00:30.000]four")
	if want := []string{"one", "two", "", "four"}; !slices.Equal(got, want) {
		t.Errorf("AlignLRC = %q, want %q", got, want)
	}
	if got := AlignLRC(lines, ""); len(got) != 4 || got[0] != "" {
		t.Errorf("without a translation: %q", got)
	}
}
