package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/app"
)

// TestWelcomeFirstRun opens the guide only in an install that had no
// settings file, walks it with Next and Back from the keyboard, and saves
// that it was done as Escape skips it, for it not to open again.
func TestWelcomeFirstRun(t *testing.T) {
	old := newTestShell(t)
	old.welcomeFirstRun()
	if old.welcome.open {
		t.Fatal("the guide opens in an install that had a settings file")
	}

	s := newTestShellOf(t, func(o *app.Options) {
		if err := os.Remove(filepath.Join(o.DataDir, "settings.json")); err != nil {
			t.Fatal(err)
		}
	})
	s.welcomeFirstRun()
	if !s.welcome.open {
		t.Fatal("the guide doesn't open in a new install")
	}
	tt := testerOf(t, s, 1280, 900)
	show(s, tt, "/")
	if !tt.HasText("Welcome to Mixyun") || !tt.HasText("Step 1 of 5") {
		t.Fatalf("the guide shows %q", tt.Texts())
	}
	// Next has the focus, for Enter; Shift+Tab reaches Back before it.
	tt.Key(0, ui.KeyEnter)
	tt.Advance(time.Second)
	if !tt.HasText("Step 2 of 5") || !tt.HasText("Follow the computer") {
		t.Fatalf("Next shows %q", tt.Texts())
	}
	tt.Key(ui.Shift, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)
	tt.Advance(time.Second)
	if !tt.HasText("Step 1 of 5") {
		t.Fatalf("Back shows %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if s.welcome.open || !s.svc.Preferences.Get().FirstRunDone {
		t.Fatalf("Escape leaves the guide open %v, done %v", s.welcome.open, s.svc.Preferences.Get().FirstRunDone)
	}
	s.welcomeFirstRun()
	if s.welcome.open {
		t.Error("the guide opens again once skipped")
	}

	// Settings › General opens it again.
	show(s, tt, "/settings/general")
	click(t, s, tt, "show the guide")
	tt.Advance(time.Second)
	if !s.welcome.open || !tt.HasText("Step 1 of 5") {
		t.Errorf("Settings › General doesn't open the guide: %q", tt.Texts())
	}
}
