package downloads

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fzn0x/mixmusic/internal/media"
)

// TestNetEaseDownloads keeps the songs of NetEase Cloud Music, which come
// as MP3 or FLAC files, by their songs' keys, and serves them as what
// they are; a Spotify track downloads its recording there.
func TestNetEaseDownloads(t *testing.T) {
	read := func(parts ...string) []byte {
		b, err := os.ReadFile(filepath.Join(parts...))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	mp3, flac := read("..", "audio", "testdata", "tone.mp3"), read("..", "..", "flac", "testdata", "stereo-16.flac")
	f := &fetcher{audio: map[string][]byte{"ne1": mp3, "ne2": flac}, mime: map[string]string{"ne1": "audio/mpeg", "ne2": "audio/flac"}}
	ne := func(id, title string) media.Track {
		return media.Track{ID: "ne:" + id, Source: media.SourceNetEase, Title: title}
	}
	mt := &matcher{matches: map[string]media.Track{"sp:tear": ne("2", "Teardrop")}}
	m := open(t, t.TempDir(), Options{Fetcher: f, Matcher: mt})
	m.Add([]media.Track{ne("1", "A song"), sp("tear", "Teardrop")}, "best")
	m.Start()
	until(t, "the downloads", func() bool { return kind(m, "ne:1") == Done && kind(m, "sp:tear") == Done })
	for id, want := range map[string][2]string{"ne:1": {"ne1.mp3", "audio/mpeg"}, "sp:tear": {"ne2.flac", "audio/flac"}} {
		k, _ := m.Lookup(id)
		if filepath.Base(k.Path) != want[0] {
			t.Errorf("%s is kept at %s, want %s", id, k.Path, want[0])
		}
		rec := httptest.NewRecorder()
		if !m.Serve(rec, httptest.NewRequest("GET", "/", nil), k.Video) || rec.Header().Get("Content-Type") != want[1] {
			t.Errorf("%s is served as %q, want %s", id, rec.Header().Get("Content-Type"), want[1])
		}
	}
}
