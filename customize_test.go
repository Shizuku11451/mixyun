package main

import (
	"slices"
	"testing"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/internal/settings"
)

// customize saves what change sets of the window's choices, and has s show
// it.
func customize(t *testing.T, s *shell, change func(c *settings.Customize)) {
	t.Helper()
	if _, err := s.svc.Preferences.Update(func(st *settings.Settings) { change(&st.Customize) }); err != nil {
		t.Fatal(err)
	}
	s.refresh()
}

// TestSourcesOff leaves the services turned off out of Home, unfetched,
// and out of Search, and hides the pivots once one service is left.
func TestSourcesOff(t *testing.T) {
	s := newTestShell(t)
	customize(t, s, func(c *settings.Customize) {
		c.SourcesOff = []string{settings.SourceSpotify, settings.SourceYouTubeMusic, settings.SourceYouTube}
	})
	tt := testerOf(t, s, 1280, 1400)
	delete(s.loads, "home:spotify")
	delete(s.loads, "home:youtube")
	show(s, tt, "/")
	for _, key := range []string{"home:spotify", "home:youtube"} {
		if _, ok := s.loads[key]; ok {
			t.Errorf("Home fetches %s, turned off", key)
		}
	}
	if _, ok := s.loads["home:netease"]; !ok {
		t.Error("Home doesn't fetch NetEase Cloud Music's shelves")
	}
	if tt.HasText("Everything") {
		t.Errorf("Home shows its pivots with one service on: %q", tt.Texts())
	}

	s.loads["search:tide"] = &loaded[app.SearchResponse]{done: true, value: app.SearchResponse{SpotifySignedIn: true,
		Spotify: media.SearchResults{Tracks: []media.Track{{ID: "sp:1", Source: media.SourceSpotify, Title: "Spotify Tide"}}},
		NetEase: media.SearchResults{Tracks: []media.Track{{ID: "ne:1", Source: media.SourceNetEase, Title: "NetEase Tide"}}},
	}}
	show(s, tt, searchPath("tide"))
	if !tt.HasText("NetEase Tide") || tt.HasText("Spotify Tide") {
		t.Errorf("Search with Spotify off shows %q", tt.Texts())
	}
	if _, ok := tt.Find("Services"); ok {
		t.Error("Search offers its services with one on")
	}
}

// TestHomePivotKept keeps the pivot of Home chosen in the settings, for
// the next launch.
func TestHomePivotKept(t *testing.T) {
	s := newTestShell(t)
	tt := testerOf(t, s, 1280, 1400)
	show(s, tt, "/")
	click(t, s, tt, "网易云音乐")
	if got := s.svc.Preferences.Get().HomePivot; got != string(media.SourceNetEase) {
		t.Errorf("HomePivot = %q, want netease", got)
	}
}

// TestRailPages leaves the pages the user hid off the sidebar, and
// Explore while YouTube Music is off; Home, Search, Library and Settings
// stay.
func TestRailPages(t *testing.T) {
	s := newTestShell(t)
	customize(t, s, func(c *settings.Customize) {
		c.SidebarHidden = []string{"stats", "home"}
		c.SourcesOff = []string{settings.SourceYouTubeMusic}
	})
	var ids []string
	for _, n := range s.railPages() {
		ids = append(ids, n.id)
	}
	if want := []string{"home", "search", "roam", "library", "marketplace", "settings"}; !slices.Equal(ids, want) {
		t.Errorf("the sidebar shows %v, want %v", ids, want)
	}
}
