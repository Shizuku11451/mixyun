package netease

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSongWiki reads a song's genre and language off its 音乐百科, as
// NetEase answered for 晴天.
func TestSongWiki(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/eapi/song/play/about/block/page" || !strings.Contains(r.Header.Get("Cookie"), "MUSIC_U=u") {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, `{"code":200,"data":{"blocks":[{"code":"SONG_PLAY_ABOUT_MUSIC_MEMORY","creatives":[]},
			{"code":"SONG_PLAY_ABOUT_SONG_BASIC","creatives":[
			{"creativeType":"songTag","uiElement":{"mainTitle":{"title":"曲风"}},"resources":[{"uiElement":{"mainTitle":{"title":"流行-华语流行"}}}]},
			{"creativeType":"songBizTag","uiElement":{"mainTitle":{"title":"推荐标签"}},"resources":[{"uiElement":{"mainTitle":{"title":"治愈"}}}]},
			{"creativeType":"language","uiElement":{"mainTitle":{"title":"语种"},"textLinks":[{"text":"国语"}]},"resources":[]}]}]}}`)
	}))
	defer srv.Close()
	c := New()
	c.eapiURL = srv.URL
	if _, _, err := c.SongWiki(context.Background(), "186016"); err != ErrSignedOut {
		t.Errorf("signed out: %v", err)
	}
	c.SetSession("u")
	genre, language, err := c.SongWiki(context.Background(), "186016")
	if err != nil || genre != "流行" || language != "华语" {
		t.Errorf("SongWiki = %q %q %v", genre, language, err)
	}
}
