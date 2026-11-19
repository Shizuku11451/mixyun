package settings

import (
	"slices"
	"testing"
)

// TestCustomizeKeepsOneSourceOn keeps a source on when settings.json turns
// every one off, and drops the keys it doesn't know.
func TestCustomizeKeepsOneSourceOn(t *testing.T) {
	var s Settings
	s.SourcesOff = append(slices.Clone(MusicSources), "nope", SourceSpotify)
	s.SidebarHidden = []string{"home", "stats", "stats"}
	s.HomePivot = "nope"
	got := s.Clean()
	if !got.SourceOn(MusicSources[0]) || len(got.SourcesOff) != len(MusicSources)-1 {
		t.Errorf("SourcesOff = %v, want every source but %s", got.SourcesOff, MusicSources[0])
	}
	if !slices.Equal(got.SidebarHidden, []string{"stats"}) || got.HomePivot != "" {
		t.Errorf("SidebarHidden = %v, HomePivot = %q", got.SidebarHidden, got.HomePivot)
	}
}
