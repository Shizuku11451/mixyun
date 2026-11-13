package netease

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRoam plays the songs of three of a tag's playlists, each once, at
// most 100 of them.
func TestRoam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/weapi/playlist/list":
			fmt.Fprint(w, `{"code":200,"playlists":[{"id":1},{"id":2},{"id":3},{"id":4}]}`)
		case "/weapi/v6/playlist/detail":
			// Every playlist holds the same 80 songs: 80 in all.
			ids := make([]string, 80)
			for i := range ids {
				ids[i] = fmt.Sprintf(`{"id":%d}`, i+1)
			}
			fmt.Fprintf(w, `{"code":200,"playlist":{"trackIds":[%s]}}`, strings.Join(ids, ","))
		case "/weapi/v3/song/detail":
			songs := make([]string, 80)
			for i := range songs {
				songs[i] = fmt.Sprintf(`{"id":%d,"name":"s"}`, i+1)
			}
			fmt.Fprintf(w, `{"code":200,"songs":[%s]}`, strings.Join(songs, ","))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := New()
	c.BaseURL = srv.URL
	tracks, err := c.Roam(context.Background(), "90后")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 80 {
		t.Errorf("Roam gave %d tracks, want the 80 songs once each", len(tracks))
	}
}

// TestRoamLiked plays what 心动模式 gives after one of the liked songs, and
// the liked songs themselves when it gives nothing.
func TestRoamLiked(t *testing.T) {
	heart := `{"code":200,"data":[{"songInfo":{"id":7,"name":"like them"}},{"songInfo":{"id":1,"name":"liked"}}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/weapi/user/playlist":
			fmt.Fprint(w, `{"code":200,"playlist":[{"id":5,"creator":{"userId":2}},{"id":10,"creator":{"userId":42}},{"id":11,"creator":{"userId":42}}]}`)
		case "/weapi/v6/playlist/detail":
			fmt.Fprint(w, `{"code":200,"playlist":{"trackIds":[{"id":1},{"id":2}]}}`)
		case "/weapi/playmode/intelligence/list":
			fmt.Fprint(w, heart)
		case "/weapi/v3/song/detail":
			fmt.Fprint(w, `{"code":200,"songs":[{"id":1,"name":"liked"},{"id":2,"name":"liked too"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := New()
	c.BaseURL = srv.URL
	tracks, err := c.RoamLiked(context.Background(), "42")
	if err != nil || len(tracks) != 2 || tracks[0].Title != "like them" {
		t.Errorf("RoamLiked gave %+v, %v", tracks, err)
	}
	heart = `{"code":200,"data":[]}`
	if tracks, err := c.RoamLiked(context.Background(), "42"); err != nil || len(tracks) != 2 || tracks[0].Title == "like them" {
		t.Errorf("without 心动模式: %+v, %v", tracks, err)
	}
}
