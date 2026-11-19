package app

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fzn0x/mixmusic/builtin"
	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/internal/settings"
)

// The song here is made up.

func openDataApp(t *testing.T, data string) *App {
	t.Helper()
	a, err := New(Options{DataDir: data, CacheDir: filepath.Join(t.TempDir(), "cache"), Builtin: builtin.FS, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// TestBackupRoundTrip backs up a library, settings and a playlist's
// cover, and imports them on a fresh data folder, where they are once
// Mixyun starts again; what they replaced is kept in before-import.
func TestBackupRoundTrip(t *testing.T) {
	from := filepath.Join(t.TempDir(), "data")
	a := openDataApp(t, from)
	song := media.Track{ID: "yt:aaaaaaaaaaa", Source: media.SourceYouTube, Title: "Slow Ferry"}
	if err := a.library.SetLiked(song, true); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Preferences{a}).Update(func(s *settings.Settings) { s.Volume = 0.37 }); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(from, "mix-covers"), 0o755)
	os.WriteFile(filepath.Join(from, "mix-covers", "mx1.png"), []byte("png"), 0o644)
	os.WriteFile(filepath.Join(from, "spotify-token.json"), []byte(`{"secret":true}`), 0o600)
	file := filepath.Join(t.TempDir(), BackupName(time.Now()))
	if err := (&System{a}).ExportBackup(file); err != nil {
		t.Fatal(err)
	}
	a.Close()
	z, err := zip.OpenReader(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		if f.Name == "spotify-token.json" {
			t.Error("the backup holds the Spotify token")
		}
	}
	z.Close()

	to := filepath.Join(t.TempDir(), "data")
	b := openDataApp(t, to)
	m, err := (&System{b}).ImportBackup(file)
	if err != nil {
		t.Fatal(err)
	}
	if m.App != backupApp || m.Version != "test" || len(m.Parts) == 0 {
		t.Errorf("manifest: %+v", m)
	}
	if b.library.IsLiked(song.ID) {
		t.Error("the import replaced the library before Mixyun started again")
	}
	b.Close()

	c := openDataApp(t, to)
	defer c.Close()
	if !c.library.IsLiked(song.ID) {
		t.Error("the liked song didn't come back")
	}
	if v := c.settings.Get().Volume; v != 0.37 {
		t.Errorf("volume %v, want 0.37", v)
	}
	if data, err := os.ReadFile(filepath.Join(to, "mix-covers", "mx1.png")); string(data) != "png" {
		t.Errorf("cover: %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(to, beforeImport, "settings.json")); err != nil {
		t.Errorf("what the import replaced wasn't kept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(to, pendingBackup)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the backup stayed pending: %v", err)
	}
}

// writeZip writes a zip of files to a new file.
func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "backup.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, text := range files {
		w, _ := zw.Create(name)
		w.Write([]byte(text))
	}
	zw.Close()
	f.Close()
	return p
}

// TestBackupBadFile refuses files that aren't whole backups of Mixyun,
// and leaves the data as it was, as Mixyun starts again too.
func TestBackupBadFile(t *testing.T) {
	manifest := `{"app":"Mixyun"}`
	text := filepath.Join(t.TempDir(), "notes.zip")
	os.WriteFile(text, []byte("not a zip"), 0o644)
	bad := map[string]string{
		"not a zip":     text,
		"no manifest":   writeZip(t, map[string]string{"library.json": "{}"}),
		"foreign":       writeZip(t, map[string]string{"manifest.json": `{"app":"Other"}`, "library.json": "{}"}),
		"broken json":   writeZip(t, map[string]string{"manifest.json": manifest, "library.json": "{"}),
		"bad settings":  writeZip(t, map[string]string{"manifest.json": manifest, "settings.json": `{"volume":`}),
		"outside parts": writeZip(t, map[string]string{"manifest.json": manifest, "../escape.json": "{}"}),
		"not a part":    writeZip(t, map[string]string{"manifest.json": manifest, "spotify-token.json": "{}"}),
	}
	data := filepath.Join(t.TempDir(), "data")
	a := openDataApp(t, data)
	song := media.Track{ID: "yt:bbbbbbbbbbb", Source: media.SourceYouTube, Title: "Tin Roof Rain"}
	if err := a.library.SetLiked(song, true); err != nil {
		t.Fatal(err)
	}
	for name, p := range bad {
		if _, err := (&System{a}).ImportBackup(p); err == nil {
			t.Errorf("%s: imported", name)
		}
	}
	if _, err := os.Stat(filepath.Join(data, pendingBackup)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a bad file is pending: %v", err)
	}
	a.Close()

	// A pending backup that went bad meanwhile changes nothing either.
	os.WriteFile(filepath.Join(data, pendingBackup), []byte("not a zip"), 0o600)
	b := openDataApp(t, data)
	defer b.Close()
	if !b.library.IsLiked(song.ID) {
		t.Error("a bad backup changed the library")
	}
}
