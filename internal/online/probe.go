package online

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	// The proxy of Windows' settings is in http.DefaultTransport before
	// probeClient clones it.
	_ "github.com/fzn0x/mixmusic/internal/sysproxy"
)

// The addresses Probe asks, and how; tests change them. YouTube's answer
// 204 with no body, where yt-dlp and YouTube Music ask; a captive portal's
// page doesn't. Spotify's API and NetEase Cloud Music's answer anything at
// all: a portal can't answer them over TLS. NetEase's is asked where the
// other two are blocked.
var (
	youtubeProbes = []string{"https://www.youtube.com/generate_204", "https://music.youtube.com/generate_204"}
	anyProbes     = []string{"https://api.spotify.com/v1/", "https://interface.music.163.com/"}
	probeClient   = newProbeClient()
)

// newProbeClient returns a client that asks once, on a connection of its
// own, and takes no redirect: a portal redirects.
func newProbeClient() *http.Client {
	c := &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		t = t.Clone()
		t.DisableKeepAlives = true
		c.Transport = t
	}
	return c
}

// Probe asks YouTube, Spotify and NetEase Cloud Music whether they answer, at once, and returns
// nil as soon as one does, or why none did.
func Probe(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	n := len(youtubeProbes) + len(anyProbes)
	answers := make(chan error, n)
	for _, u := range youtubeProbes {
		go func() { answers <- ask(ctx, u, http.StatusNoContent) }()
	}
	for _, u := range anyProbes {
		go func() { answers <- ask(ctx, u, 0) }()
	}
	var errs []error
	for range n {
		err := <-answers
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// ask asks u, and returns nil when it answers status, or anything when
// status is 0.
func ask(ctx context.Context, u string, status int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	res, err := probeClient.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	if status != 0 && res.StatusCode != status {
		return fmt.Errorf("%s answered HTTP %d", u, res.StatusCode)
	}
	return nil
}
