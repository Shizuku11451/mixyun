package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/settings"
)

// TestAddSettings keeps the parts of a place in their order, whatever the
// order they register in, and refuses two parts at one order, and a place
// in no section.
func TestAddSettings(t *testing.T) {
	var parts []settingsPart
	for _, order := range []int{30, 10, 20, 15} {
		parts = withPart(parts, "playback", order, func(*shell, *ui.Context) {})
	}
	for i, want := range []int{10, 15, 20, 30} {
		if parts[i].order != want {
			t.Fatalf("the parts are at %v", parts)
		}
	}
	panics := func(name string, add func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s doesn't panic", name)
			}
		}()
		add()
	}
	panics("a part at a taken order", func() { withPart(parts, "playback", 20, nil) })
	panics("a part in no section", func() { addSettings("nowhere › Sound", 10, nil) })
	if _, ok := settingsParts["nowhere › Sound"]; ok {
		t.Error("a part in no section registered")
	}
	for place, parts := range settingsParts {
		for i := 1; i < len(parts); i++ {
			if parts[i-1].order >= parts[i].order {
				t.Errorf("the parts of %s are out of order", place)
			}
		}
	}
}

// TestSettingsOrder shows the groups and rows that register from their
// features' files in their sections, in order.
func TestSettingsOrder(t *testing.T) {
	s := newTestShell(t)
	tt := testerOf(t, s, 1280, 4400)
	for path, texts := range map[string][]string{
		"/settings/appearance": {"layout", "Layout", "theme", "Follow the computer", "mix options", "colour and type",
			"Colours from a file", "Follow the artwork", "Accent colour", "Zoom", "Text size", "Density", "Icons only in the sidebar"},
		"/settings/playback": {"sound", "Quality", "Crossfade", "Autoplay", "speed", "skip parts of videos"},
		"/settings/audio":    {"output", "Output device", "Even loudness", "Volume levelling", "Play 5.1 surround when a track has it", "equalizer"},
		"/settings/content": {"spotify tracks", "How Spotify tracks play", "explicit content", "songs to skip", "Skip songs made with AI",
			"Skip songs you dislike", "music videos"},
		"/settings/lyrics":  {"Translate lyrics to", "Lyrics sources", "lyrics style"},
		"/settings/storage": {"downloads", "offline cache", "music on this computer", "tools"},
	} {
		show(s, tt, path)
		var above float32
		for _, text := range texts {
			r, ok := tt.Find(text)
			if !ok {
				t.Errorf("%s doesn't show %q", path, text)
				continue
			}
			if r.Y <= above {
				t.Errorf("%s shows %q above what comes before it", path, text)
			}
			above = r.Y
		}
	}
}

// openAdvanced opens the fold of what few change at the foot of each
// section of the settings.
func openAdvanced(t *testing.T, s *shell) {
	t.Helper()
	if _, err := s.svc.Preferences.Update(func(st *settings.Settings) { st.AdvancedSettings = true }); err != nil {
		t.Fatal(err)
	}
	s.refresh()
}

// TestAdvancedFold folds what few change at a section's foot, opens it on
// a click, which the settings keep, and has the search find a folded row
// and open the fold for it.
func TestAdvancedFold(t *testing.T) {
	s := newTestShell(t)
	tt := testerOf(t, s, 1280, 2400)
	show(s, tt, "/settings/playback")
	if !tt.HasText("Advanced") || tt.HasText("Volume curve") {
		t.Fatalf("the fold doesn't fold the volume curve: %q", tt.Texts())
	}
	click(t, s, tt, "Advanced")
	if !tt.HasText("Volume curve") || !s.svc.Preferences.Get().AdvancedSettings {
		t.Fatalf("opening the fold shows %q, keeps %v", tt.Texts(), s.svc.Preferences.Get().AdvancedSettings)
	}
	click(t, s, tt, "Advanced")
	if tt.HasText("Volume curve") || s.svc.Preferences.Get().AdvancedSettings {
		t.Fatal("closing the fold leaves it open")
	}

	show(s, tt, settingsPath("", "volume curve"))
	if !tt.HasText("Volume curve") {
		t.Fatalf("the search doesn't find the folded row: %q", tt.Texts())
	}
	click(t, s, tt, "Show volume in playback")
	tt.Advance(time.Second)
	if got := s.router.Location(); got != "/settings/playback" || !tt.HasText("Volume curve") {
		t.Errorf("show in went to %q, showing %q", got, tt.Texts())
	}
}

// TestSettingsSections lists the sections at the page's left in a wide
// window, and as pivots over it in a narrow one.
func TestSettingsSections(t *testing.T) {
	s := newTestShell(t)
	tt := testerOf(t, s, 1600, 1000)
	show(s, tt, "/settings")
	if _, ok := tt.AccessByRole(ui.RoleTab, "audio"); !ok {
		t.Fatalf("no section audio at the left: %q", tt.Texts())
	}
	click(t, s, tt, "audio")
	tt.Advance(time.Second)
	if got := s.router.Location(); got != "/settings/audio" || !tt.HasText("Output device") {
		t.Errorf("audio went to %q", got)
	}
	narrow := testerOf(t, s, 640, 1000)
	show(s, narrow, "/settings")
	if _, ok := narrow.AccessByRole(ui.RoleTab, "audio"); ok || !narrow.HasText("audio") {
		t.Errorf("a narrow window shows %q", narrow.Texts())
	}
}
