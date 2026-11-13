package lyrics

import (
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
