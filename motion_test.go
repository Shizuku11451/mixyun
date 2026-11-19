package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/settings"
	"github.com/fzn0x/mixmusic/locales"
)

func TestPageEntry(t *testing.T) {
	for _, c := range []struct {
		from, to  string
		back, rtl bool
		want      ui.Pose
		moves     bool
	}{
		{"/", "/search", false, false, ui.Pose{Y: 10}, true},
		{"/search", "/settings", false, false, ui.Pose{Y: 10}, true},
		{"/album/1", "/library", false, false, ui.Pose{Y: 10}, true},
		{"/search", "/album/1", false, false, ui.Pose{X: 24}, true},
		{"/album/1", "/artist/2", false, false, ui.Pose{X: 24}, true},
		{"/search", "/album/1", false, true, ui.Pose{X: -24}, true},
		{"/album/1", "/search", true, false, ui.Pose{X: -24}, true},
		{roomsPath, roomsPath + "/7", false, false, ui.Pose{X: 24}, true},
		{"/library/liked", "/library/downloads", false, false, ui.Pose{}, false},
		{"/settings", "/settings/accounts", false, false, ui.Pose{}, false},
	} {
		got, moves := pageEntry(c.from, c.to, c.back, c.rtl)
		if got != c.want || moves != c.moves {
			t.Errorf("pageEntry(%q, %q, back %v, rtl %v) = %+v, %v; want %+v, %v", c.from, c.to, c.back, c.rtl, got, moves, c.want, c.moves)
		}
	}
}

// empty is a text of the stats page while nothing was counted.
const empty = "Nothing counted yet"

// TestPanelGrowsAndSettles opens and closes the queue in a layout that
// moves: the panel grows from the window's edge, and shrinks back to it,
// and the window asks for no frame once it's still.
func TestPanelGrowsAndSettles(t *testing.T) {
	s, tt := themed(t, "mix", settings.LayoutExpressive)
	title := s.lk.words(locales.PanelUpNext())
	s.togglePanel("queue")
	tt.Frame()
	tt.Advance(50 * time.Millisecond)
	mid, _ := tt.Find(title)
	tt.Advance(time.Second)
	end, ok := tt.Find(title)
	if !ok || mid.X <= end.X {
		t.Errorf("the panel's title is at %v on the way, and at %v once open", mid.X, end.X)
	}
	settled(t, tt)
	s.togglePanel("queue")
	tt.Frame()
	tt.Advance(50 * time.Millisecond)
	if closing, ok := tt.Find(title); !ok || closing.X <= end.X {
		t.Errorf("closing, the panel's title is at %v, %v", closing.X, ok)
	}
	tt.Advance(time.Second)
	if tt.HasText(title) {
		t.Error("the closed panel still shows")
	}
	settled(t, tt)
}

// TestPanelOpenAtLaunch shows a panel the settings restore at once.
func TestPanelOpenAtLaunch(t *testing.T) {
	s := newTestShell(t)
	st := s.svc.Preferences.Get()
	st.Layout = settings.LayoutExpressive
	if _, err := s.svc.Preferences.Set(st); err != nil {
		t.Fatal(err)
	}
	s.refresh()
	s.panel = "queue"
	tt := tester(t, s)
	if r, ok := tt.Find(s.lk.words(locales.PanelUpNext())); !ok || r.W == 0 || tt.FramePending() {
		t.Errorf("the panel there at launch shows at %v, %v, asking for frames %v", r, ok, tt.FramePending())
	}
}

// TestPageRisesAndSettles goes to another page in a layout that moves:
// it comes in from a little below, and the window settles.
func TestPageRisesAndSettles(t *testing.T) {
	s, tt := themed(t, "mix", settings.LayoutExpressive)
	s.navigate("/stats")
	tt.Frame()
	tt.Advance(16 * time.Millisecond)
	mid, _ := tt.Find(empty)
	tt.Advance(time.Second)
	end, ok := tt.Find(empty)
	if !ok || mid.Y <= end.Y {
		t.Errorf("the page's text is at %v as it comes in, and at %v once in", mid.Y, end.Y)
	}
	settled(t, tt)
}

// TestStageOpensAndCloses opens the full-screen lyrics in a layout that
// moves, and closes them: nothing asks for frames once they're still.
func TestStageOpensAndCloses(t *testing.T) {
	s, tt := themed(t, "mix", settings.LayoutExpressive)
	s.stage.open = true
	tt.Frame()
	tt.Advance(16 * time.Millisecond)
	mid, _ := tt.AccessByRole(ui.RoleDialog, locales.LyricsFullScreen())
	tt.Advance(time.Second)
	end, ok := tt.AccessByRole(ui.RoleDialog, locales.LyricsFullScreen())
	if !ok || mid.Bounds.Y <= end.Bounds.Y {
		t.Errorf("the stage is at %v as it opens, and at %v (%v) once open", mid.Bounds, end.Bounds, ok)
	}
	s.stage.open = false
	tt.Frame()
	tt.Advance(time.Second)
	settled(t, tt)
}
