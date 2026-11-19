package audio

import (
	"testing"
	"time"
)

// nativeTrack is a 44.1 kHz FLAC of noise, and its samples.
func nativeTrack(t *testing.T, n int) (string, [][2]int16) {
	s := make([][2]int16, n)
	x := uint32(7)
	for i := range s {
		for c := range 2 {
			x = x*1664525 + 1013904223
			s[i][c] = int16(x >> 17)
		}
	}
	return flacAt(t, 44100, s), s
}

// waitRate waits for the device to play at rate.
func (f *fakeDevice) waitRate(t *testing.T, rate int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		f.mu.Lock()
		r := f.rate
		f.mu.Unlock()
		if r == rate {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the device plays at %d Hz, want %d", r, rate)
		}
		time.Sleep(time.Millisecond)
	}
}

// TestNativeRate plays a 44.1 kHz FLAC, which SetNative lets play at its
// own rate, on a device started at 44.1 kHz, its samples as they are.
func TestNativeRate(t *testing.T) {
	const n = 20000
	path, samples := nativeTrack(t, n)
	e, dev, _ := newTestEngine(t)
	e.SetNative([]int{44100})
	e.Load(path, Cue{})
	buffered(t, e, 0, ringFrames/2)
	e.Play()
	dev.waitRate(t, 44100)
	got := dev.play(t, n/2)
	// The output starts again at 44.1 kHz: what plays from then on, past
	// the fade in, is the file's samples, in order, from where it took up.
	at := -1
	for i := 0; i+1 < n; i++ {
		if got[2*(n/2-1)] == float32(samples[i][0])/32768 && got[2*(n/2-1)+1] == float32(samples[i][1])/32768 {
			at = i - (n/2 - 1)
			break
		}
	}
	if at < 0 {
		t.Fatal("the last frame played isn't one of the file's: it was converted")
	}
	for f := n/2 - 1000; f < n/2; f++ {
		for c := range 2 {
			if want := float32(samples[at+f][c]) / 32768; got[2*f+c] != want {
				t.Fatalf("frame %d, channel %d: %v, want %v", f, c, got[2*f+c], want)
			}
		}
	}
	if rate, err := e.NativeRate(); rate != 44100 || err != nil {
		t.Errorf("NativeRate() = %d, %v; want 44100, nil", rate, err)
	}
}

// TestNativeRateFallsBack plays the track at 48 kHz when the device fails
// to start at its own rate, and tells why.
func TestNativeRateFallsBack(t *testing.T) {
	path, _ := nativeTrack(t, 20000)
	e, dev, _ := newTestEngine(t)
	dev.noRate = 44100
	e.SetNative([]int{44100})
	e.Load(path, Cue{})
	buffered(t, e, 0, ringFrames/2)
	e.Play()
	dev.waitRate(t, Rate)
	deadline := time.Now().Add(5 * time.Second)
	for {
		rate, err := e.NativeRate()
		if rate == Rate && err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("NativeRate() = %d, %v; want %d and why", rate, err, Rate)
		}
		time.Sleep(time.Millisecond)
	}
	if l, _ := e.Loaded(); l.Rate != Rate {
		t.Errorf("the track decodes at %d Hz, want %d", l.Rate, Rate)
	}
	dev.play(t, 4800) // and plays
}
