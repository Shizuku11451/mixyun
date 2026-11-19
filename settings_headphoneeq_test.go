package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
)

// TestHeadphoneEQImport imports a profile with a line it can't run, which
// tells the line and keeps what played, then an AutoEq one, which plays
// and offers to turn its boost down.
func TestHeadphoneEQImport(t *testing.T) {
	dir := t.TempDir()
	bad, good := filepath.Join(dir, "bad.txt"), filepath.Join(dir, "HD 600 ParametricEQ.txt")
	os.WriteFile(bad, []byte("Preamp: -3 dB\nFilter 1: ON BP Fc 100 Hz Gain 3 dB Q 1\n"), 0o644)
	os.WriteFile(good, []byte("Preamp: -1 dB\nFilter 1: ON PK Fc 1000 Hz Gain 6 dB Q 1\n"), 0o644)
	file := bad
	dialog := openFileDialog
	openFileDialog = func(mygo.OpenDialogOptions) ([]string, error) { return []string{file}, nil }
	t.Cleanup(func() { openFileDialog = dialog })

	s := newTestShell(t)
	tt := tester(t, s)
	show(s, tt, "/settings/audio")
	click(t, s, tt, "Choose a ParametricEQ.txt file")
	until(t, s, tt, "the bad line told", func() bool {
		return tt.HasText("Line 2 has a filter other than PK, LSC or HSC: Filter 1: ON BP Fc 100 Hz Gain 3 dB Q 1 The profile that plays is unchanged.")
	})
	if s.set.HeadphoneEQ.Profile != "" {
		t.Fatal("a failed import changed the profile")
	}

	file = good
	click(t, s, tt, "Choose a ParametricEQ.txt file")
	until(t, s, tt, "the profile", func() bool { return s.set.HeadphoneEQ.Enabled && s.set.HeadphoneEQ.Name == "HD 600 ParametricEQ" })
	until(t, s, tt, "its boost", func() bool { return tt.HasText("Turn down by -5.0 dB") })
	click(t, s, tt, "Turn down by -5.0 dB")
	until(t, s, tt, "turned down", func() bool { return s.set.HeadphoneEQ.Attenuation == -5 })
	if tt.HasText("Line 2 has a filter") {
		t.Error("the old problem still shows")
	}
}
