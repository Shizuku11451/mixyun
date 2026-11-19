package audio

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// rangedSource is a file played from frame start to end, as a CUE sheet's
// track is: "path#start-end".
type rangedSource struct {
	Source
	start, end time.Duration
}

func (r rangedSource) Range() (time.Duration, time.Duration) { return r.start, r.end }

func rangedOpener(ctx context.Context, url string, interrupted func() bool) (Source, error) {
	path, part, _ := strings.Cut(url, "#")
	src, err := fileOpener(ctx, path, interrupted)
	if err != nil || part == "" {
		return src, err
	}
	var a, b int64
	fmt.Sscanf(part, "%d-%d", &a, &b)
	return rangedSource{src, duration(a), duration(b)}, nil
}

func rangedEngine(t *testing.T) (*Engine, *fakeDevice, *recorder) {
	e, dev, rec := newTestEngine(t)
	e.open = rangedOpener
	return e, dev, rec
}

// TestRangedPlaysItsPart plays the parts of an album that a CUE sheet
// names, of one file: each its own frames, from its start, its length its
// own, ending at its end; a seek stays within it.
func TestRangedPlaysItsPart(t *testing.T) {
	const n, split = 30000, 3*4096 + 1234 + quantum/3
	whole, _, _ := album(t, n, split)
	ref, dev, _ := newTestEngine(t)
	ref.Load(whole, Cue{})
	buffered(t, ref, 0, ringFrames/2)
	ref.Play()
	want := dev.play(t, n)

	for _, tc := range []struct {
		url      string
		from, to int
	}{
		{fmt.Sprintf("%s#0-%d", whole, split), 0, split},
		{fmt.Sprintf("%s#%d-0", whole, split), split, n},
	} {
		e, dev, rec := rangedEngine(t)
		if err := e.Load(tc.url, Cue{}); err != nil {
			t.Fatal(err)
		}
		buffered(t, e, 0, ringFrames/2)
		if d, _ := e.decks[0].info(); frames(d) != int64(tc.to-tc.from) {
			t.Errorf("%s: %d frames long, want %d", tc.url, frames(d), tc.to-tc.from)
		}
		e.Play()
		got := dev.play(t, tc.to-tc.from+2000)
		rec.waitFor(t, "ended")
		for i := 2 * fadeFrames; i < len(got); i++ {
			w := float32(0)
			if j := 2*tc.from + i; i < 2*(tc.to-tc.from) {
				w = want[j]
			}
			if got[i] != w {
				t.Fatalf("%s: frame %d: %v, want %v", tc.url, i/2, got[i], w)
			}
		}
	}

	// A seek is within the part: 0.1 s into the second is 0.1 s past the
	// split of the whole.
	e, dev, _ := rangedEngine(t)
	e.Load(fmt.Sprintf("%s#%d-0", whole, split), Cue{})
	e.Seek(100 * time.Millisecond)
	buffered(t, e, 0, 4000)
	e.Play()
	got := dev.play(t, 4000)
	at := split + int(frames(100*time.Millisecond))
	for i := 2 * fadeFrames; i < len(got); i++ {
		if got[i] != want[2*at+i] {
			t.Fatalf("after the seek, frame %d: %v, want %v", i/2, got[i], want[2*at+i])
		}
	}
}

// TestRangedGapless runs one part of the file into the next as #21 runs a
// file into the next: the album plays whole, sample for sample.
func TestRangedGapless(t *testing.T) {
	const n, split = 30000, 3*4096 + 1234 + quantum/3
	whole, _, _ := album(t, n, split)
	ref, dev, _ := newTestEngine(t)
	ref.Load(whole, Cue{})
	buffered(t, ref, 0, ringFrames/2)
	ref.Play()
	want := dev.play(t, n-2000)

	a, b := fmt.Sprintf("%s#0-%d", whole, split), fmt.Sprintf("%s#%d-0", whole, split)
	e, dev, rec := rangedEngine(t)
	e.Load(a, Cue{})
	e.Preload(b)
	e.Follow(b)
	buffered(t, e, 0, ringFrames/2)
	buffered(t, e, 1, ringFrames/2)
	e.Play()
	got := dev.play(t, n-2000)
	rec.waitFor(t, "ended")
	for i := 2 * fadeFrames; i < len(want); i++ {
		if got[i] != want[i] {
			t.Fatalf("frame %d (the join at %d): %v, want %v", i/2, split, got[i], want[i])
		}
	}
}
