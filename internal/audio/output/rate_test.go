package output

import (
	"errors"
	"runtime"
	"slices"
	"testing"
)

// TestRangeRates tells the standard rates in the ranges Core Audio tells
// of a device: single rates, a continuous range, and none.
func TestRangeRates(t *testing.T) {
	for _, c := range []struct {
		ranges [][2]float64
		want   []int
	}{
		{[][2]float64{{96000, 96000}, {44100, 44100}, {48000, 48000}}, []int{44100, 48000, 96000}},
		{[][2]float64{{8000, 192000}}, []int{44100, 48000, 88200, 96000, 176400, 192000}},
		{[][2]float64{{44099.9, 44100.1}, {32000, 32000}}, []int{44100}},
		{nil, nil},
	} {
		if got := rangeRates(c.ranges); !slices.Equal(got, c.want) {
			t.Errorf("rates of %v: %v, want %v", c.ranges, got, c.want)
		}
	}
	if !rateIn([][2]float64{{32000, 32000}}, 32000) {
		t.Error("32000 not in a range of 32000")
	}
}

// TestRate opens no stream at a rate other than Rate on the default device,
// which no system changes the rate of, and tells no rates of a device that
// isn't there; it opens no device.
func TestRate(t *testing.T) {
	if _, err := Open(Options{Rate: 44100}); !errors.Is(err, ErrRate) {
		t.Errorf("Open at 44100 Hz: %v, want ErrRate", err)
	}
	want := ErrUnsupported
	if runtime.GOOS == "darwin" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") {
		want = ErrNoDevice
	}
	if _, err := DeviceRates("no such device"); !errors.Is(err, want) {
		t.Errorf("DeviceRates: %v, want %v", err, want)
	}
}
