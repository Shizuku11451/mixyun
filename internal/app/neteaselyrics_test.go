package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzn0x/mixmusic/internal/match"
	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/internal/netease"
)

// TestNetEaseTranslation shows NetEase Cloud Music's lyrics word by word
// with its own translation into Chinese, timed as they are, and leaves
// other languages, and songs it has no translation of, to Unison.
func TestNetEaseTranslation(t *testing.T) {
	translation := `"tlyric":{"lyric":"[00:01.000]第一行\n[00:03.000]第二行"},"ytlrc":{"lyric":"[00:01.200]第一行\n[00:03.200]第二行"},`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/weapi/song/lyric/v1" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"code":200,%s"lrc":{"lyric":"[00:01.000]first line\n[00:03.000]second line"},
			"yrc":{"lyric":"[1200,900](1200,400,0)first (1600,500,0)line\n[3200,900](3200,400,0)second (3600,500,0)line"}}`, translation)
	}))
	defer srv.Close()
	a := &App{netease: netease.New(), matcher: &match.Matcher{}}
	a.netease.BaseURL = srv.URL
	c := &Catalog{a: a}
	track := media.Track{ID: "ne:1", Source: media.SourceNetEase}
	ctx := context.Background()

	l, ok := c.neteaseLyrics(ctx, track)
	if !ok || len(l.Lines) != 2 || len(l.Lines[0].Words) != 2 || l.Lines[1].TimeMs != 3200 {
		t.Fatalf("lyrics: %+v", l)
	}
	tr, ok := c.neteaseTranslation(ctx, track, l, "zh")
	if !ok || len(tr.Lines) != 2 || tr.Lines[0].Text != "第一行" || tr.Lines[1].Text != "第二行" {
		t.Errorf("translation: %+v, %v", tr, ok)
	}
	if _, ok := c.neteaseTranslation(ctx, track, l, "en"); ok {
		t.Error("NetEase translated into English")
	}
	translation = ""
	if _, ok := c.neteaseTranslation(ctx, track, l, "zh"); ok {
		t.Error("a song without a translation has one")
	}
}
