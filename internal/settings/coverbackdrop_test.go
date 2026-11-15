package settings

import "testing"

// TestCoverBackdropVeil keeps the veil of the cover's backdrop from 0 to
// 0.9: a value outside, or none, is the default's.
func TestCoverBackdropVeil(t *testing.T) {
	d := Default()
	if !d.CoverBackdrop || d.CoverBackdropVeil != 0.55 {
		t.Fatalf("default: %v, %v", d.CoverBackdrop, d.CoverBackdropVeil)
	}
	for v, want := range map[float64]float64{0: 0, 0.3: 0.3, 0.9: 0.9, -0.1: 0.55, 1: 0.55} {
		s := Default()
		s.CoverBackdropVeil = v
		if got := s.Clean().CoverBackdropVeil; got != want {
			t.Errorf("veil %v cleans to %v, want %v", v, got, want)
		}
	}
	s, err := Decode([]byte(`{"theme": "mix"}`))
	if err != nil || !s.CoverBackdrop || s.CoverBackdropVeil != 0.55 {
		t.Errorf("a file from before it: %v, %v, %v", s.CoverBackdrop, s.CoverBackdropVeil, err)
	}
}
