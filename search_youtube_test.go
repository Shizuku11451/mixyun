package main

import (
	"testing"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/internal/media"
)

// searchYouTubeShell has a search whose YouTube scope found a tutorial,
// which YouTube Music's catalog doesn't list.
func searchYouTubeShell(t *testing.T) *shell {
	s := newTestShell(t)
	songs := []media.Track{
		{ID: "yt:song", Source: media.SourceYouTube, Title: "Boiler Room Anthem", Artists: []media.Ref{{Name: "Someone"}}},
	}
	web := []media.Track{
		{ID: "yt:tutorial", Source: media.SourceYouTube, Title: "How to mix vinyl", Artists: []media.Ref{{Name: "A Channel"}}, Plays: 1_200_000, DurationMs: 600_000, Video: true},
	}
	s.loads["search:vinyl"] = &loaded[app.SearchResponse]{done: true, value: app.SearchResponse{
		YouTube: media.SearchResults{Tracks: songs},
	}}
	s.loads["searchweb:vinyl"] = &loaded[media.SearchResults]{done: true, value: media.SearchResults{Videos: web}}
	return s
}

// TestSearchYouTubeScope shows YouTube's own videos in the YouTube scope,
// and only those.
func TestSearchYouTubeScope(t *testing.T) {
	s := searchYouTubeShell(t)
	s.searchScope = youtubeScope
	tt := testerOf(t, s, 1280, 1800)
	show(s, tt, searchPath("vinyl"))
	if !showsText(tt.Texts(), "How to mix vinyl") {
		t.Errorf("the YouTube scope doesn't show the tutorial: %q", tt.Texts())
	}
	if showsText(tt.Texts(), "Boiler Room Anthem") {
		t.Error("the YouTube scope shows a YouTube Music song")
	}
	// YouTube's own search lists videos only, so the pivots don't apply.
	if showsText(tt.Texts(), "Podcasts") {
		t.Error("the YouTube scope shows the pivots")
	}
}

// TestSearchScopesNameYouTube checks the scope the search page offers.
func TestSearchScopesNameYouTube(t *testing.T) {
	scopes := searchScopeLabels()
	for i, want := range map[int]string{spotifyScope: "Spotify", neteaseScope: "网易云音乐", youtubeMusicScope: "YouTube Music", youtubeScope: "YouTube"} {
		if scopes[i] != want {
			t.Errorf("scope %d is %q, want %q", i, scopes[i], want)
		}
	}
	if len(scopes) != 5 {
		t.Errorf("scopes = %v", scopes)
	}
}
