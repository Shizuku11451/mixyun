package main

import (
	"testing"

	"github.com/fzn0x/mixmusic/internal/app"
)

// TestRoamPage shows the Roam page of the rail: a tile for each tag, by
// era, language and genre, and one for what the user liked once they
// signed in to NetEase Cloud Music.
func TestRoamPage(t *testing.T) {
	s := newTestShell(t)
	tt := testerOf(t, s, 1280, 900)
	show(s, tt, "/roam")
	for _, label := range []string{"Roaming 90后…", "Roaming 粤语…", "Roaming 摇滚…"} {
		if _, ok := tt.Find(label); !ok {
			t.Errorf("no tile labelled %q", label)
		}
	}
	if _, ok := tt.Find("Roaming Your likes…"); ok {
		t.Error("signed out, a tile roams the likes")
	}
	s.loads["accounts"] = &loaded[app.AccountsStatus]{done: true, value: app.AccountsStatus{NetEase: app.NetEaseAccount{SignedIn: true}}}
	show(s, tt, "/roam")
	if _, ok := tt.Find("Roaming Your likes…"); !ok {
		t.Error("signed in, no tile roams the likes")
	}
}
