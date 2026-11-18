package app

import (
	"slices"
	"testing"

	"github.com/fzn0x/mixmusic/internal/media"
)

// TestMergeSuggestions takes NetEase's and YouTube Music's in turn,
// NetEase's first, and one of the searches that differ only in case.
func TestMergeSuggestions(t *testing.T) {
	a, b, c := media.Item{Kind: "a"}, media.Item{Kind: "b"}, media.Item{Kind: "c"}
	got := mergeSuggestions(
		media.Suggestions{Searches: []string{"周杰伦", "Jay Chou", "晴天"}, Items: []media.Item{a, c}},
		media.Suggestions{Searches: []string{"jay chou", "jay chou songs"}, Items: []media.Item{b}},
	)
	if want := []string{"周杰伦", "jay chou", "jay chou songs", "晴天"}; !slices.Equal(got.Searches, want) {
		t.Errorf("searches %q, want %q", got.Searches, want)
	}
	if kinds := []media.ItemKind{got.Items[0].Kind, got.Items[1].Kind, got.Items[2].Kind}; !slices.Equal(kinds, []media.ItemKind{"a", "b", "c"}) {
		t.Errorf("items %v", kinds)
	}
}
