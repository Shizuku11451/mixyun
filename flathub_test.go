package main

import (
	"strings"
	"testing"

	"github.com/fzn0x/mixmusic/internal/app"
)

// Flathub's build updates through the software center: About names it, and
// tells where a new version comes from, without a download.
func TestFlathubUpdates(t *testing.T) {
	if got, _ := buildName("flathub"); got != "Flathub" {
		t.Errorf("About calls Flathub's build %q", got)
	}
	u := app.UpdateStatus{Kind: "flathub", AutomaticChecks: true}
	if got := newVersionDoes(u); !strings.Contains(got, "Flathub") || !strings.Contains(got, "software center") || strings.Contains(got, "download") {
		t.Errorf("the checks say a new version %q", got)
	}
	got := replacing("flathub", "")
	for _, want := range []string{"software center", "Flathub", "flatpak update", "Your library and settings stay."} {
		if !strings.Contains(got, want) {
			t.Errorf("a new version of Flathub's build says %q, without %q", got, want)
		}
	}
	if strings.Contains(got, ".flatpak") || strings.Contains(got, "flatpak install") {
		t.Errorf("Flathub's build is offered a bundle: %q", got)
	}
	// The release's own .flatpak still offers its download.
	if got := newVersionDoes(app.UpdateStatus{Kind: "flatpak"}); got != " A new version offers its download." {
		t.Errorf("the release's .flatpak says a new version %q", got)
	}
}
