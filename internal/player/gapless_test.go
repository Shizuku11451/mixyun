package player

import (
	"testing"
	"time"

	"github.com/fzn0x/mixmusic/internal/media"
)

// TestGaplessFollow has the engine run a file on this computer into the
// next one (#21), and follow nothing once the queue no longer has it next.
func TestGaplessFollow(t *testing.T) {
	h := newHarness(t, Saved{})
	local := tracks("a", "b", "c")
	for i := range local {
		local[i].Source = media.SourceFile
	}
	h.p.PlayTracks(local, 0)
	h.wait("a playing", playing("a"))
	h.p.Time(11*time.Second, 200*time.Second)
	h.wait("b followed", func(*Player) bool { return h.stream.has("follow url:yt:b") })
	h.p.RemoveAt(1)
	h.wait("b no longer followed", func(*Player) bool { return h.stream.has("follow ") })
}

// TestGaplessNotStreams leaves a stream to play after the track before it
// as it did: only files follow.
func TestGaplessNotStreams(t *testing.T) {
	h := newHarness(t, Saved{})
	h.p.PlayTracks(tracks("a", "b"), 0)
	h.wait("a playing", playing("a"))
	h.p.Time(11*time.Second, 200*time.Second)
	h.wait("b preloaded", func(*Player) bool { return h.stream.has("preload url:yt:b") })
	for _, c := range h.stream.copy() {
		if len(c) >= 6 && c[:6] == "follow" {
			t.Errorf("a stream was followed: %q", c)
		}
	}
}
