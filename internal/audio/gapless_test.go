package audio

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// flacOf writes a FLAC file of 16-bit stereo samples at 48 kHz, in frames
// of 4096 samples but the last, of verbatim subframes.
func flacOf(t *testing.T, samples [][2]int16) string {
	t.Helper()
	const block = 4096
	info := make([]byte, 34)
	binary.BigEndian.PutUint16(info[0:], block)
	binary.BigEndian.PutUint16(info[2:], block)
	binary.BigEndian.PutUint64(info[10:], uint64(48000)<<44|1<<41|15<<36|uint64(len(samples)))
	b := append([]byte("fLaC\x80\x00\x00\x22"), info...)
	for i := 0; i*block < len(samples); i++ {
		part := samples[i*block : min((i+1)*block, len(samples))]
		frame := []byte{0xFF, 0xF8, 0xCA, 0x18, byte(i)}
		if len(part) != block {
			frame[2] = 0x7A
			frame = binary.BigEndian.AppendUint16(frame, uint16(len(part)-1))
		}
		frame = append(frame, crc8(frame))
		for c := range 2 {
			frame = append(frame, 0x02) // verbatim
			for _, s := range part {
				frame = binary.BigEndian.AppendUint16(frame, uint16(s[c]))
			}
		}
		b = append(b, binary.BigEndian.AppendUint16(frame, crc16(frame))...)
	}
	path := filepath.Join(t.TempDir(), "track.flac")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func crc8(b []byte) byte {
	var c byte
	for _, v := range b {
		c ^= v
		for range 8 {
			c = c<<1 ^ 0x07*(c>>7)
		}
	}
	return c
}

func crc16(b []byte) uint16 {
	var c uint16
	for _, v := range b {
		c ^= uint16(v) << 8
		for range 8 {
			c = c<<1 ^ 0x8005*(c>>15)
		}
	}
	return c
}

// album is a continuous signal, noise that tells each frame from its
// neighbours, split into two tracks at split, which isn't at the end of a
// FLAC frame nor of the engine's quantum: the whole, and its two parts.
func album(t *testing.T, n, split int) (whole, a, b string) {
	s := make([][2]int16, n)
	x := uint32(1)
	for i := range s {
		for c := range 2 {
			x = x*1664525 + 1013904223
			s[i][c] = int16(x >> 17)
		}
	}
	return flacOf(t, s), flacOf(t, s[:split]), flacOf(t, s[split:])
}

// gaplessEngine plays a, with b preloaded on the other deck and filled.
func gaplessEngine(t *testing.T, a, b string, follow bool) (*Engine, *fakeDevice, *recorder) {
	e, dev, rec := newTestEngine(t)
	e.Load(a, Cue{})
	e.Preload(b)
	if follow {
		e.Follow(b)
	}
	buffered(t, e, 0, ringFrames/2)
	buffered(t, e, 1, ringFrames/2)
	e.Play()
	return e, dev, rec
}

// TestGapless runs a track into the next as it ends, frame by frame: what
// plays is the album whole, with no frame added, passed over, repeated or
// faded at the join, and the next track plays on as Load takes it.
func TestGapless(t *testing.T) {
	const n, split = 30000, 3*4096 + 1234 + quantum/3
	whole, a, b := album(t, n, split)

	ref, dev, _ := newTestEngine(t)
	ref.Load(whole, Cue{})
	buffered(t, ref, 0, ringFrames/2)
	ref.Play()
	want := dev.play(t, n-2000)

	e, dev, rec := gaplessEngine(t, a, b, true)
	got := dev.play(t, split+4000)
	rec.waitFor(t, "ended")
	if err := e.Load(b, Cue{Autoplay: true}); err != nil {
		t.Fatal(err)
	}
	got = append(got, dev.play(t, n-2000-split-4000)...)
	for i := 2 * fadeFrames; i < len(want); i++ {
		if got[i] != want[i] {
			t.Fatalf("frame %d (the join at %d): %v, want %v", i/2, split, got[i], want[i])
		}
	}
	if rec.has("playing false") {
		t.Error("it paused at the join")
	}
}

// TestGaplessNotFollowed plays a track preloaded that Follow no longer
// names, as the queue changed near the end, only once Load asks for it.
func TestGaplessNotFollowed(t *testing.T) {
	const n, split = 20000, 9000
	_, a, b := album(t, n, split)
	e, dev, rec := gaplessEngine(t, a, b, true)
	e.Follow("")
	got := dev.play(t, split+2000)
	rec.waitFor(t, "ended")
	for i := 2 * split; i < len(got); i++ {
		if got[i] != 0 {
			t.Fatalf("frame %d after the end: %v, want silence", i/2, got[i])
		}
	}
	// Load forgets what it followed: the track loaded in place of the one
	// before ends as it would.
	e, _, _ = gaplessEngine(t, a, b, true)
	e.Load(a, Cue{Autoplay: true})
	e.mu.Lock()
	follow := e.follow
	e.mu.Unlock()
	if follow != "" {
		t.Errorf("Load kept %q to follow", follow)
	}
}
