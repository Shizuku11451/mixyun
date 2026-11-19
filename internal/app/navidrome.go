package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fzn0x/mixmusic/internal/listenbrainz"
	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/internal/stream"
	"github.com/fzn0x/mixmusic/internal/subsonic"
)

// navidromeSecret is the key, in the system's credential store, of the
// Navidrome connection: the server, the user's name and password.
const navidromeSecret = "navidrome"

// navidromeLogin connects MixMusic to the user's Navidrome server.
type navidromeLogin struct {
	Server   string `json:"server"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// NavidromeAccount is the state of the Navidrome connection, without the
// password.
type NavidromeAccount struct {
	SignedIn bool   `json:"signedIn"`
	Server   string `json:"server,omitempty"`
	User     string `json:"user,omitempty"`
}

// openNavidrome connects the client with the login kept, if any.
func (a *App) openNavidrome() {
	if l, err := a.navidromeStore.Load(); err == nil && l.Server != "" {
		a.navidrome.SetLogin(l.Server, l.User, l.Password)
	}
	a.resolver.Navidrome = a.navidromeStream
}

// navidromeAccount is the state of the connection.
func (a *App) navidromeAccount() NavidromeAccount {
	if a.navidrome == nil { // an App of the tests
		return NavidromeAccount{}
	}
	server, user := a.navidrome.Login()
	return NavidromeAccount{SignedIn: server != "", Server: server, User: user}
}

// NavidromeLogin connects to the user's Navidrome server, at the address
// server as they wrote it, as user: the server checks the password, and
// MixMusic keeps it in the system's credential store until the user
// disconnects. Plain http goes only to a server on the local network.
func (s *Accounts) NavidromeLogin(ctx context.Context, server, user, password string) (AccountsStatus, error) {
	a := s.a
	user = strings.TrimSpace(user)
	if strings.TrimSpace(server) == "" || user == "" || password == "" {
		return AccountsStatus{}, errors.New("Navidrome needs its address, your user name and your password")
	}
	addr, plain, err := listenbrainz.ParseServer(server)
	if err != nil {
		return AccountsStatus{}, errors.New("this isn't the address of a server: write it as https://music.example.com")
	}
	if plain && !listenbrainz.Local(ctx, addr, nil) {
		return AccountsStatus{}, errPlainRemote
	}
	check := subsonic.New()
	check.SetLogin(addr, user, password)
	if err := check.Ping(ctx); err != nil {
		return AccountsStatus{}, fmt.Errorf("Mixyun couldn't connect to Navidrome, so nothing was kept: %w", err)
	}
	l := navidromeLogin{Server: addr, User: user, Password: password}
	if err := a.navidromeStore.Save(l); err != nil {
		return AccountsStatus{}, err
	}
	a.navidrome.SetLogin(l.Server, l.User, l.Password)
	notify()
	return a.accountsStatus(ctx), nil
}

// NavidromeLogout forgets the Navidrome connection.
func (s *Accounts) NavidromeLogout(ctx context.Context) error {
	s.a.navidrome.SetLogin("", "", "")
	err := s.a.navidromeStore.Clear()
	notify()
	return err
}

// navidromeStream finds the audio of a Navidrome song: its original file,
// whose size and type the server tells.
//
// shortcut: always the original file, whatever Settings › Quality says;
// ask the server to transcode (format, maxBitRate) when people want to
// save data on it.
func (a *App) navidromeStream(ctx context.Context, songID string) (stream.Stream, error) {
	song, err := a.navidrome.Song(ctx, songID)
	if err != nil {
		return stream.Stream{}, err
	}
	u, err := a.navidrome.StreamURL(songID)
	if err != nil {
		return stream.Stream{}, err
	}
	// The address doesn't expire: a while keeps the resolver from asking
	// the server again at each seek.
	return stream.Stream{URL: u, MimeType: song.ContentType, Size: song.Size, Duration: time.Duration(song.Duration) * time.Second,
		Bitrate: float64(song.BitRate), Expires: time.Now().Add(6 * time.Hour)}, nil
}

// NavidromeAlbums returns the albums of the user's Navidrome library.
func (c *Collection) NavidromeAlbums(ctx context.Context) ([]media.Album, error) {
	return c.a.navidrome.Albums(ctx)
}

// NavidromeArtists returns the artists of the user's Navidrome library.
func (c *Collection) NavidromeArtists(ctx context.Context) ([]media.Artist, error) {
	return c.a.navidrome.Artists(ctx)
}
