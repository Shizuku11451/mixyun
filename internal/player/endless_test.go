package player

import (
	"context"
	"slices"
	"testing"

	"github.com/fzn0x/mixmusic/internal/media"
)

// TestPlayEndless has an endless queue go on with more of its own where
// it runs out, again and again, in the radio's place; a queue played
// after it goes on with its radio.
func TestPlayEndless(t *testing.T) {
	h := newHarness(t, Saved{}, func(f *fakeServices) {
		f.settings.Autoplay = true
		f.radio = []media.Track{track("r1", "Radio")}
	})
	n := 0
	more := func(context.Context) ([]media.Track, error) {
		n++
		// The song it gave already isn't queued twice.
		return []media.Track{track("a", "Artist a"), track("m"+string(rune('0'+n)), "More")}, nil
	}
	h.p.PlayEndless(tracks("a"), more)
	h.wait("a playing", playing("a"))
	h.p.Ended()
	h.wait("m1 playing", playing("m1"))
	h.p.Ended()
	h.wait("m2 playing", playing("m2"))
	if got := queueIDs(h.p.State()); !slices.Equal(got, []string{"a", "m1*", "m2*"}) {
		t.Errorf("the queue is %v", got)
	}

	h.p.PlayTracks(tracks("b"), 0)
	h.wait("b playing", playing("b"))
	h.p.Ended()
	h.wait("r1 playing", playing("r1"))
	if n != 2 {
		t.Errorf("more was asked %d times, the last for a queue of others", n)
	}
}
