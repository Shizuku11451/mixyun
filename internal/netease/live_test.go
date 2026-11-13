package netease

import (
	"context"
	"os"
	"testing"
)

// TestLive asks NetEase itself, signed out: MIXMUSIC_LIVE=1 runs it.
func TestLive(t *testing.T) {
	if os.Getenv("MIXMUSIC_LIVE") == "" {
		t.Skip("MIXMUSIC_LIVE=1 asks NetEase Cloud Music")
	}
	ctx := context.Background()
	c := New()
	res, err := c.Search(ctx, "周杰伦 晴天")
	if err != nil || len(res.Tracks) == 0 {
		t.Fatalf("Search: %v, %d tracks", err, len(res.Tracks))
	}
	t.Logf("first: %s by %s (%s), %d albums %d artists %d playlists", res.Tracks[0].Title, res.Tracks[0].ArtistNames(), res.Tracks[0].ID, len(res.Albums), len(res.Artists), len(res.Playlists))
	if _, err := c.Playlist(ctx, "3778678", ""); err != nil { // the hot songs chart
		t.Fatalf("Playlist: %v", err)
	}
	if ly, err := c.Lyrics(ctx, "186016"); err != nil || ly.LRC == "" {
		t.Fatalf("Lyrics: %v", err)
	}
	// Sung word by word, and translated.
	if ly, err := c.Lyrics(ctx, "1859245776"); err != nil || ly.YRC == "" || ly.Translation == "" || ly.YRCTranslation == "" {
		t.Fatalf("Lyrics, word by word: %v, %d %d %d bytes", err, len(ly.YRC), len(ly.Translation), len(ly.YRCTranslation))
	}
	if _, err := c.Similar(ctx, "186016"); err != nil {
		t.Fatalf("Similar: %v", err)
	}
	if shelves, err := c.Home(ctx, ""); err != nil || len(shelves) == 0 {
		t.Fatalf("Home: %v", err)
	}
	if roam, err := c.Roam(ctx, RoamTags[0][1]); err != nil || len(roam) == 0 {
		t.Fatalf("Roam: %v, %d tracks", err, len(roam))
	}
	if sug, err := c.Suggestions(ctx, "周杰"); err != nil || len(sug.Searches) == 0 || len(sug.Items) == 0 {
		t.Fatalf("Suggestions: %v, %d searches %d items", err, len(sug.Searches), len(sug.Items))
	} else {
		t.Logf("suggestions: %v, %d items", sug.Searches, len(sug.Items))
	}
	// A free song: signed out, NetEase gives the most it may of each level.
	for _, level := range []string{"standard", "exhigh", "lossless"} {
		a, err := c.SongAudio(ctx, "1859245776", level)
		t.Logf("audio at %s: %s, %d bytes, %v", level, a.Type, a.Size, err)
	}
	// 音乐百科 answers only a session: MIXMUSIC_NETEASE_U is its MUSIC_U.
	if u := os.Getenv("MIXMUSIC_NETEASE_U"); u != "" {
		c.SetSession(u)
		me, err := c.Me(ctx)
		if err != nil {
			t.Fatalf("Me: %v", err)
		}
		if liked, err := c.RoamLiked(ctx, me.ID); err != nil || len(liked) == 0 {
			t.Fatalf("RoamLiked: %v, %d tracks", err, len(liked))
		} else {
			t.Logf("roaming the likes: %d tracks, first %s", len(liked), liked[0].Title)
		}
		if genre, language, err := c.SongWiki(ctx, "186016"); err != nil || genre == "" || language == "" {
			t.Fatalf("SongWiki: %q %q %v", genre, language, err)
		} else {
			t.Logf("wiki: %s, %s", genre, language)
		}
	}
}
