// Package subsonic reads a personal music server through the Subsonic API,
// as Navidrome speaks it: its artists, albums and songs, and the audio of
// songs as their original files.
package subsonic

import (
	"cmp"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/fzn0x/mixmusic/internal/media"
)

// ErrSignedOut is returned while no server is set.
var ErrSignedOut = errors.New("not connected to Navidrome")

// Client asks one Subsonic server, as one user.
type Client struct {
	HTTP *http.Client

	mu                     sync.Mutex
	server, user, password string
}

// New returns a client of no server.
func New() *Client { return &Client{HTTP: &http.Client{Timeout: 20 * time.Second}} }

// SetLogin sets the server's address and the user's name and password; an
// empty server signs out.
func (c *Client) SetLogin(server, user, password string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.server, c.user, c.password = server, user, password
}

// Login returns the server's address and the user's name, "" while signed
// out.
func (c *Client) Login() (server, user string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.server, c.user
}

// endpoint returns the address of a method of the API with params and the
// user's credentials: a salted token of the password, never the password.
func (c *Client) endpoint(method string, params url.Values) (string, error) {
	c.mu.Lock()
	server, user, password := c.server, c.user, c.password
	c.mu.Unlock()
	if server == "" {
		return "", ErrSignedOut
	}
	salt := make([]byte, 8)
	rand.Read(salt)
	s := hex.EncodeToString(salt)
	sum := md5.Sum([]byte(password + s))
	q := url.Values{"u": {user}, "t": {hex.EncodeToString(sum[:])}, "s": {s}, "v": {"1.16.1"}, "c": {"Mixyun"}, "f": {"json"}}
	for k, v := range params {
		q[k] = v
	}
	return server + "/rest/" + method + "?" + q.Encode(), nil
}

// get calls a method of the API and decodes its answer into out. Its
// errors never quote the address, which carries the credentials.
func (c *Client) get(ctx context.Context, method string, params url.Values, out any) error {
	u, err := c.endpoint(method, params)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return errors.New("navidrome: bad server address")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return fmt.Errorf("Navidrome doesn't answer: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Navidrome answered HTTP %d", resp.StatusCode)
	}
	var body struct {
		R json.RawMessage `json:"subsonic-response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.R == nil {
		return errors.New("this isn't a Navidrome server, or its Subsonic API")
	}
	var status struct {
		Status string `json:"status"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body.R, &status); err != nil {
		return err
	}
	if status.Status != "ok" {
		if status.Error != nil {
			return fmt.Errorf("Navidrome: %s (error %d)", status.Error.Message, status.Error.Code)
		}
		return errors.New("Navidrome refused the request")
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body.R, out)
}

// Ping checks that the server answers and takes the credentials.
func (c *Client) Ping(ctx context.Context) error { return c.get(ctx, "ping", nil, nil) }

// Song is a song as the server lists it.
type Song struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Album       string `json:"album"`
	AlbumID     string `json:"albumId"`
	Artist      string `json:"artist"`
	ArtistID    string `json:"artistId"`
	Track       int    `json:"track"`
	DiscNumber  int    `json:"discNumber"`
	Duration    int    `json:"duration"` // in seconds
	BitRate     int    `json:"bitRate"`  // in kbit/s
	Size        int64  `json:"size"`
	Suffix      string `json:"suffix"`
	ContentType string `json:"contentType"`
}

type album struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Artist    string `json:"artist"`
	ArtistID  string `json:"artistId"`
	Year      int    `json:"year"`
	SongCount int    `json:"songCount"`
	Song      []Song `json:"song"`
}

type artist struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Album []album `json:"album"`
}

func ref(id, name string) media.Ref {
	if id == "" {
		return media.Ref{Name: name}
	}
	return media.Ref{ID: media.ID(media.SourceNavidrome, id), Name: name}
}

// MediaTrack returns the song as a track.
//
// shortcut: no artwork, as the server's cover art addresses carry the
// credentials; serve covers through the local proxy to show them.
func (s Song) MediaTrack() media.Track {
	t := media.Track{ID: media.ID(media.SourceNavidrome, s.ID), Source: media.SourceNavidrome, Title: s.Title,
		Artists: []media.Ref{ref(s.ArtistID, s.Artist)}, DurationMs: s.Duration * 1000, Artwork: []media.Image{}}
	if s.Album != "" {
		a := ref(s.AlbumID, s.Album)
		t.Album = &a
	}
	return t
}

func (a album) album() media.Album {
	out := media.Album{ID: media.ID(media.SourceNavidrome, a.ID), Source: media.SourceNavidrome, Title: a.Name,
		Artists: []media.Ref{ref(a.ArtistID, a.Artist)}, Kind: media.AlbumKindAlbum, Artwork: []media.Image{},
		TrackCount: a.SongCount, Tracks: []media.Track{}}
	if a.Year > 0 {
		out.Year = strconv.Itoa(a.Year)
	}
	// In disc and track order, whatever order the server lists them in.
	songs := slices.Clone(a.Song)
	slices.SortStableFunc(songs, func(x, y Song) int {
		return cmp.Or(cmp.Compare(x.DiscNumber, y.DiscNumber), cmp.Compare(x.Track, y.Track))
	})
	for _, s := range songs {
		out.Tracks = append(out.Tracks, s.MediaTrack())
	}
	return out
}

func (a artist) artist() media.Artist {
	out := media.Artist{ID: media.ID(media.SourceNavidrome, a.ID), Source: media.SourceNavidrome, Name: a.Name,
		Artwork: []media.Image{}, TopTracks: []media.Track{}, Albums: []media.Album{}, Singles: []media.Album{}, Related: []media.Ref{}}
	for _, al := range a.Album {
		out.Albums = append(out.Albums, al.album())
	}
	return out
}

// Artists returns every artist of the library, by name.
func (c *Client) Artists(ctx context.Context) ([]media.Artist, error) {
	var r struct {
		Artists struct {
			Index []struct {
				Artist []artist `json:"artist"`
			} `json:"index"`
		} `json:"artists"`
	}
	if err := c.get(ctx, "getArtists", nil, &r); err != nil {
		return nil, err
	}
	out := []media.Artist{}
	for _, ix := range r.Artists.Index {
		for _, a := range ix.Artist {
			out = append(out, a.artist())
		}
	}
	return out, nil
}

// Artist returns an artist with their albums.
func (c *Client) Artist(ctx context.Context, id string) (media.Artist, error) {
	var r struct {
		Artist artist `json:"artist"`
	}
	if err := c.get(ctx, "getArtist", url.Values{"id": {id}}, &r); err != nil {
		return media.Artist{}, err
	}
	return r.Artist.artist(), nil
}

// Album returns an album with its songs.
func (c *Client) Album(ctx context.Context, id string) (media.Album, error) {
	var r struct {
		Album album `json:"album"`
	}
	if err := c.get(ctx, "getAlbum", url.Values{"id": {id}}, &r); err != nil {
		return media.Album{}, err
	}
	return r.Album.album(), nil
}

// Albums returns the library's albums, by name.
//
// shortcut: the first 500 alone, the most the API gives at once; page
// through with offset once a library's albums shelf needs them all.
func (c *Client) Albums(ctx context.Context) ([]media.Album, error) {
	var r struct {
		List struct {
			Album []album `json:"album"`
		} `json:"albumList2"`
	}
	if err := c.get(ctx, "getAlbumList2", url.Values{"type": {"alphabeticalByName"}, "size": {"500"}}, &r); err != nil {
		return nil, err
	}
	out := []media.Album{}
	for _, a := range r.List.Album {
		out = append(out, a.album())
	}
	return out, nil
}

// Song returns a song, with what its file is.
func (c *Client) Song(ctx context.Context, id string) (Song, error) {
	var r struct {
		Song Song `json:"song"`
	}
	err := c.get(ctx, "getSong", url.Values{"id": {id}}, &r)
	return r.Song, err
}

// StreamURL returns the address of a song's original file, untranscoded.
// It carries the credentials: it goes to the local proxy alone, never to
// the log or the clipboard.
func (c *Client) StreamURL(id string) (string, error) {
	return c.endpoint("stream", url.Values{"id": {id}, "format": {"raw"}})
}
