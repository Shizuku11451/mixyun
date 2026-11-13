package netease

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzn0x/mixmusic/internal/media"
)

// TestSuggestions gives NetEase's searches, then two of each kind at most
// in its order, the songs read again for their covers; and the searches
// alone when the items' page fails.
func TestSuggestions(t *testing.T) {
	failItems := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/weapi/search/suggest/keyword":
			fmt.Fprint(w, `{"code":200,"result":{"allMatch":[{"keyword":"周杰伦"},{"keyword":"周杰伦晴天"}]}}`)
		case "/weapi/search/suggest/web":
			if failItems {
				http.Error(w, "no", http.StatusInternalServerError)
				return
			}
			fmt.Fprint(w, `{"code":200,"result":{"order":["artists","songs","albums"],
				"songs":[{"id":1,"name":"a"},{"id":2,"name":"b"},{"id":3,"name":"c"}],
				"artists":[{"id":6452,"name":"周杰伦"}],
				"albums":[{"id":9,"name":"叶惠美"}],
				"playlists":[{"id":7,"name":"not in the order"}]}}`)
		case "/weapi/v3/song/detail":
			fmt.Fprint(w, `{"code":200,"songs":[{"id":1,"name":"a","al":{"id":9,"picUrl":"https://p/x.jpg"}},{"id":2,"name":"b"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := New()
	c.BaseURL = srv.URL
	got, err := c.Suggestions(context.Background(), "周杰")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Searches) != 2 || got.Searches[1] != "周杰伦晴天" {
		t.Errorf("searches: %v", got.Searches)
	}
	kinds := ""
	for _, it := range got.Items {
		kinds += string(it.Kind) + " "
	}
	if kinds != "artist track track album " {
		t.Errorf("items: %s", kinds)
	}
	if len(got.Items) > 1 && (got.Items[1].Kind != media.ItemTrack || len(got.Items[1].Track.Artwork) == 0) {
		t.Errorf("the first song came without its cover: %+v", got.Items[1].Track)
	}
	failItems = true
	if got, err := c.Suggestions(context.Background(), "周杰"); err != nil || len(got.Searches) != 2 || len(got.Items) != 0 {
		t.Errorf("without the items' page: %+v, %v", got, err)
	}
}
