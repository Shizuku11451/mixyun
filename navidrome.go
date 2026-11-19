package main

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/locales"
)

// navidromeAccount connects to the user's Navidrome server with its
// address, their name and password, and disconnects.
func (s *shell) navidromeAccount(c *ui.Context, nd app.NavidromeAccount) {
	s.group(c, "Navidrome", func() {
		if nd.SignedIn {
			s.row(c, locales.AccountSettingsSignedInAs(nd.User), nd.Server, func() {
				if s.button(c, "navidrome-out", locales.AccountSettingsDisconnect(), false).Clicked() {
					s.account("", "", func(ctx context.Context) error { return s.svc.Accounts.NavidromeLogout(ctx) })
				}
			}, "sign out, log out, disconnect")
			return
		}
		// The form shows whole, as its button needs its fields.
		s.found(c, locales.AccountSettingsNavidromeConnect(), locales.AccountSettingsNavidromeForm(), func() {
			s.row(c, locales.AccountSettingsNavidromeServer(), "", func() {
				ui.TextInput(c.Key("navidrome-server"), &s.navidromeServer).Placeholder("https://music.example.com").Width(260).Label(locales.AccountSettingsNavidromeServer())
			})
			s.row(c, locales.AccountSettingsNavidromeUser(), "", func() {
				ui.TextInput(c.Key("navidrome-user"), &s.navidromeUser).Width(260).Label(locales.AccountSettingsNavidromeUser())
			})
			s.row(c, locales.AccountSettingsNavidromePassword(), "", func() {
				ui.TextInput(c.Key("navidrome-password"), &s.navidromePassword).Password().Width(260).Label(locales.AccountSettingsNavidromePassword())
			})
			busy := s.signingIn == "navidrome"
			label := locales.AccountSettingsNavidromeConnectButton()
			if busy {
				label = locales.AccountSettingsNavidromeWaiting()
			}
			ui.Row(c).Padding(12, 0).Children(func() {
				server, user, password := strings.TrimSpace(s.navidromeServer), strings.TrimSpace(s.navidromeUser), s.navidromePassword
				if s.button(c, "navidrome-in", label, true).Disabled(busy || server == "" || user == "" || password == "").Clicked() {
					s.account("navidrome", locales.AccountSettingsNavidromeConnected(), func(ctx context.Context) error {
						_, err := s.svc.Accounts.NavidromeLogin(ctx, server, user, password)
						return err
					})
				}
			})
			s.hint(c, locales.AccountSettingsNavidromeHint())
		})
	})
}

// navidromeLibrary shows the albums and the artists of the user's Navidrome
// library, whose pages play their songs.
func (s *shell) navidromeLibrary(c *ui.Context) {
	col := s.svc.Collection
	none := func(context.Context) ([]media.Playlist, error) { return nil, nil }
	noArtists := func(context.Context) ([]media.Artist, error) { return nil, nil }
	s.serviceLibrary(c, "navidrome",
		fetch(s, "navidrome:playlists", none),
		fetch(s, "navidrome:albums", col.NavidromeAlbums),
		// Every artist of the library, not those followed, under a title
		// of its own.
		fetch(s, "navidrome:artists", noArtists),
		func(c *ui.Context) {
			artists := fetch(s, "navidrome:all-artists", col.NavidromeArtists)
			switch {
			case !artists.done:
				s.loading(c)
			case artists.err != nil:
				s.problem(c, artists.err, func() { s.forget("navidrome:all-artists") })
			case len(artists.value) > 0:
				s.shelf(c, "navidrome-artists", locales.LibraryArtists(), func() {
					s.nearShelfTiles(c, "navidrome-artists", len(artists.value), func(i int) {
						s.artistTile(c, "navidrome-artist-"+count(i), artists.value[i])
					})
				})
			}
		})
}
