package main

import (
	"path/filepath"
	"testing"
)

// TestDefaultDownloadsDir keeps the downloads where each system keeps an
// app's data, and with a portable copy in its own folder.
func TestDefaultDownloadsDir(t *testing.T) {
	root := t.TempDir()
	at := func(parts ...string) string { return filepath.Join(append([]string{root}, parts...)...) }
	for _, c := range []struct {
		name, goos, portable, data, cache, xdg, want string
	}{
		{"windows", "windows", "", at("Roaming", "Mixyun"), at("Local", "Mixyun"), "", at("Local", "Mixyun", "downloads")},
		{"macOS", "darwin", "", at("Library", "Application Support", "Mixyun"), at("Library", "Caches", "Mixyun"), "",
			at("Library", "Application Support", "Mixyun", "downloads")},
		{"linux", "linux", "", at(".config", "Mixyun"), at(".cache", "Mixyun"), "", at(".local", "share", "Mixyun", "downloads")},
		{"XDG_DATA_HOME", "linux", "", at(".config", "Mixyun"), at(".cache", "Mixyun"), at("data"), at("data", "Mixyun", "downloads")},
		{"a relative XDG_DATA_HOME", "linux", "", at(".config", "Mixyun"), at(".cache", "Mixyun"), "data",
			at(".local", "share", "Mixyun", "downloads")},
		{"portable", "windows", at("portable"), at("portable", "data"), at("portable", "cache"), "", at("portable", "downloads")},
	} {
		if got := defaultDownloadsDir(c.goos, c.portable, c.data, c.cache, root, c.xdg); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}
