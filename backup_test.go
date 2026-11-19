package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"

	"github.com/fzn0x/mixmusic/internal/app"
)

// TestBackupSettings exports a backup from Settings › General, imports it
// back, which asks first and starts Mixyun again, and refuses a file that
// isn't one, without starting again.
func TestBackupSettings(t *testing.T) {
	var restarts atomic.Int32
	was := restartApp
	restartApp = func(*app.System) { restarts.Add(1) }
	t.Cleanup(func() { restartApp = was })
	dir := t.TempDir()
	file := filepath.Join(dir, "backup.zip")
	savesTo(t, file)
	chosen := file
	dialog := openFileDialog
	openFileDialog = func(mygo.OpenDialogOptions) ([]string, error) { return []string{chosen}, nil }
	t.Cleanup(func() { openFileDialog = dialog })

	s := newTestShell(t)
	tt := tester(t, s)
	show(s, tt, settingsPath("general", ""))
	click(t, s, tt, "export")
	until(t, s, tt, "the backup", func() bool { return tt.HasText("Saved the backup as backup.zip") })
	if _, err := os.Stat(file); err != nil {
		t.Fatal(err)
	}

	click(t, s, tt, "import")
	until(t, s, tt, "the question", func() bool { return tt.HasText("import and restart") })
	click(t, s, tt, "import and restart")
	until(t, s, tt, "the restart", func() bool { return restarts.Load() == 1 })

	chosen = filepath.Join(dir, "notes.zip")
	os.WriteFile(chosen, []byte("not a zip"), 0o644)
	click(t, s, tt, "import")
	until(t, s, tt, "the question", func() bool { return tt.HasText("import and restart") })
	click(t, s, tt, "import and restart")
	until(t, s, tt, "the refusal", func() bool {
		return slices.ContainsFunc(tt.Texts(), func(text string) bool { return strings.Contains(text, "isn't a backup of Mixyun") })
	})
	if restarts.Load() != 1 {
		t.Error("a bad file started Mixyun again")
	}
}
