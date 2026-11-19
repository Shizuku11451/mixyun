package subsonic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAlbum signs each request with a salted token, never the password,
// and lists an album's songs in disc and track order.
func TestAlbum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		sum := md5.Sum([]byte("secret" + q.Get("s")))
		if q.Get("u") != "me" || q.Get("t") != hex.EncodeToString(sum[:]) || q.Get("p") != "" || strings.Contains(r.URL.RawQuery, "secret") {
			fmt.Fprint(w, `{"subsonic-response":{"status":"failed","error":{"code":40,"message":"Wrong username or password"}}}`)
			return
		}
		if r.URL.Path != "/rest/getAlbum" || q.Get("id") != "al1" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"subsonic-response":{"status":"ok","album":{"id":"al1","name":"A","artist":"B","artistId":"ar1","year":2001,"song":[
			{"id":"s3","title":"Three","discNumber":2,"track":1},
			{"id":"s2","title":"Two","discNumber":1,"track":2},
			{"id":"s1","title":"One","discNumber":1,"track":1}]}}}`)
	}))
	defer srv.Close()
	c := New()
	c.SetLogin(srv.URL, "me", "secret")
	a, err := c.Album(context.Background(), "al1")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, tr := range a.Tracks {
		ids = append(ids, tr.ID)
	}
	if got := strings.Join(ids, " "); got != "nd:s1 nd:s2 nd:s3" || a.Year != "2001" || a.Artists[0].ID != "nd:ar1" {
		t.Fatalf("album = %+v, tracks %s", a, got)
	}

	c.SetLogin(srv.URL, "me", "wrong")
	if err := c.Ping(context.Background()); err == nil || !strings.Contains(err.Error(), "Wrong username") {
		t.Fatalf("ping with a wrong password = %v", err)
	}
}

// TestUnreachable tells the server doesn't answer without quoting its
// address, which carries the credentials.
func TestUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	c := New()
	c.SetLogin(srv.URL, "me", "secret")
	err := c.Ping(context.Background())
	if err == nil || strings.Contains(err.Error(), "t=") || strings.Contains(err.Error(), srv.URL) {
		t.Fatalf("ping of a closed server = %v", err)
	}
}
