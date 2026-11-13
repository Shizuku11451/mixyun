package main

import (
	"testing"

	"github.com/egoist/mygo"
)

// TestTaskbarLyricsPlace puts the lyrics over a taskbar at the bottom or
// the top, and nowhere for one at a side or hidden.
func TestTaskbarLyricsPlace(t *testing.T) {
	b := mygo.Rectangle{Width: 1920, Height: 1080}
	for _, tc := range []struct {
		work mygo.Rectangle
		want mygo.Rectangle
		ok   bool
	}{
		{mygo.Rectangle{Width: 1920, Height: 1032}, mygo.Rectangle{X: 1240, Y: 1032, Width: 360, Height: 48}, true},
		{mygo.Rectangle{Y: 40, Width: 1920, Height: 1040}, mygo.Rectangle{X: 1240, Width: 360, Height: 40}, true},
		{mygo.Rectangle{X: 60, Width: 1860, Height: 1080}, mygo.Rectangle{}, false},
		{b, mygo.Rectangle{}, false},
	} {
		got, ok := taskbarLyricsPlace([]mygo.Display{{Bounds: b, WorkArea: tc.work, Primary: true}})
		if got != tc.want || ok != tc.ok {
			t.Errorf("work area %v: %v %v, not %v %v", tc.work, got, ok, tc.want, tc.ok)
		}
	}
}
