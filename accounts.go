package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/locales"
)

func init() { addSettings("accounts", 10, (*shell).accountSettings) }

// accountSettings sign in and out of Spotify, NetEase Cloud Music and Last.fm.
func (s *shell) accountSettings(c *ui.Context) {
	st := s.accounts()
	s.spotifyAccount(c, st.Spotify)
	s.neteaseAccount(c, st.NetEase)
	s.youtubeAccount(c, st.YouTube)
	s.lastfmAccount(c, st.Lastfm)
}

func (s *shell) spotifyAccount(c *ui.Context, sp app.SpotifyAccount) {
	s.group(c, "Spotify", func() {
		if sp.SignedIn && sp.Error == "" {
			who := locales.AccountSettingsSignedIn()
			if sp.Profile != nil {
				who = locales.AccountSettingsSignedInAs(sp.Profile.Name)
			}
			s.row(c, who, locales.AccountSettingsSpotifyDesc(), func() {
				if s.button(c, "spotify-out", locales.AccountSettingsSignOut(), false).Clicked() {
					s.signOutOfSpotify()
				}
			}, "sign out, log out")
			return
		}
		if sp.Error != "" {
			s.found(c, locales.AccountSettingsSpotifyRefused(), sp.Error, func() {
				s.alert(c, locales.AccountSettingsSpotifyRefused(), sp.Error, nil).Margin(12, 0)
			})
		}
		// The form shows whole, as its button needs its field.
		s.found(c, locales.AccountSettingsSpotifySignIn(), locales.AccountSettingsSpotifyForm(), func() {
			if sp.RedirectURI == "" {
				sp.RedirectURI = s.svc.System.Info().RedirectURI
			}
			s.steps(c,
				func() {
					ui.Text(c, locales.AccountSettingsSpotifyStep1())
					s.linkOf(c, locales.AccountSettingsSpotifyDashboard(), "https://developer.spotify.com/dashboard")
					ui.Text(c, locales.AccountSettingsSpotifyStep1Tail())
				},
				func() {
					ui.Text(c, locales.AccountSettingsSpotifyRedirect())
					ui.Text(c, sp.RedirectURI).Font(monospace())
					ui.Text(c, " ")
					if s.linkOf(c, locales.AccountSettingsCopy(), "").Clicked() {
						c.WriteClipboard(sp.RedirectURI)
						s.notes = append(s.notes, notice(locales.AccountSettingsCopiedRedirect()))
					}
				},
				func() {
					ui.Text(c, locales.AccountSettingsSpotifyUsers())
				},
				func() { ui.Text(c, locales.AccountSettingsSpotifyPaste()) },
			)
			var clientID *draft
			s.row(c, locales.AccountsClientID(), "", func() {
				var in ui.Element
				in, clientID = s.settingInput(c, "client-id", s.set.SpotifyClientID, func(id string) {
					s.set.SpotifyClientID = id
					s.save()
				})
				in.Placeholder(locales.AccountSettingsCharsPlaceholder()).Width(260).Label(locales.AccountsClientID())
			})
			busy := s.signingIn == "spotify"
			label := locales.AccountSettingsSignInSpotify()
			if busy {
				label = locales.AccountSettingsWaitingBrowser()
			}
			ui.Row(c).Gap(8).Padding(12, 0).Children(func() {
				// The Client ID's draft, which its row's control made, unless
				// the search of the settings left the row out.
				id := s.set.SpotifyClientID
				if clientID == nil {
					clientID = s.drafts["client-id"]
				}
				if clientID != nil {
					id = strings.TrimSpace(clientID.text)
				}
				if s.button(c, "spotify-in", label, true).Disabled(busy || id == "").Clicked() {
					if id != s.set.SpotifyClientID {
						s.set.SpotifyClientID = id
						s.save()
					}
					s.notes = append(s.notes, notice(locales.AccountSettingsFinishSpotify()))
					s.account("spotify", locales.AccountSettingsSignedInSpotify(), func(ctx context.Context) error {
						_, err := s.svc.Accounts.SpotifyLogin(ctx)
						return err
					})
				}
				// A sign-in Spotify refused stays until the user drops it: every
				// request of Spotify fails meanwhile.
				if sp.SignedIn && s.button(c, "spotify-out", locales.AccountSettingsSignOut(), false).Disabled(busy).Clicked() {
					s.signOutOfSpotify()
				}
			})
			s.hint(c, locales.AccountSettingsSpotifyHint())
		})
	})
}

// signOutOfSpotify asks, then has MixMusic forget the Spotify sign-in.
func (s *shell) signOutOfSpotify() {
	s.ask(locales.AccountSettingsSpotifySignOutTitle(), locales.AccountSettingsSpotifySignOutBody(), locales.AccountSettingsSignOut(), func() {
		s.account("", "", func(context.Context) error { return s.svc.Accounts.SpotifyLogout() })
	})
}

// youtubeAccount signs out of a YouTube Music session kept from before
// NetEase Cloud Music took its place, which nothing else can: its
// playlists stay in the menus while it lasts. A new install has none.
func (s *shell) youtubeAccount(c *ui.Context, yt app.YouTubeAccount) {
	if !yt.SignedIn {
		return
	}
	s.group(c, "YouTube Music", func() {
		who := locales.AccountSettingsSignedIn()
		if yt.Account != nil {
			who = locales.AccountSettingsSignedInAs(yt.Account.Name)
		}
		s.row(c, who, locales.AccountSettingsYouTubeSignedInDesc(), func() {
			if s.button(c, "youtube-out", locales.AccountSettingsSignOut(), false).Clicked() {
				s.ask(locales.AccountSettingsYouTubeSignOutTitle(), locales.AccountSettingsYouTubeSignOutBody(), locales.AccountSettingsSignOut(), func() {
					s.account("", "", func(ctx context.Context) error { return s.svc.Accounts.YouTubeLogout(ctx) })
				})
			}
		}, "sign out, log out")
	})
}

func (s *shell) lastfmAccount(c *ui.Context, lf app.LastfmAccount) {
	s.group(c, "Last.fm", func() {
		if lf.Connected {
			if lf.Expired {
				title, again := locales.AccountSettingsLastfmGone(), locales.AccountSettingsLastfmAgain()
				s.found(c, title, again, func() { s.alert(c, title, again, nil).Margin(12, 0) })
			}
			s.row(c, locales.AccountSettingsScrobblingAs(lf.Name), locales.AccountSettingsScrobbleDesc(), func() {
				if s.switchOf(c.Key("scrobbling"), &s.set.Scrobbling).Label(locales.AccountSettingsScrobbleSwitch()).Changed() {
					s.save()
				}
			})
			s.found(c, locales.AccountSettingsDisconnect(), "sign out log out of Last.fm", func() {
				ui.Row(c).Padding(12, 0).Children(func() {
					if s.button(c, "lastfm-out", locales.AccountSettingsDisconnect(), false).Clicked() {
						s.ask(locales.AccountSettingsDisconnectAskTitle(), locales.AccountSettingsDisconnectAskBody(), locales.AccountSettingsDisconnect(), func() {
							s.account("", "", func(ctx context.Context) error { return s.svc.Accounts.LastfmLogout(ctx) })
						})
					}
				})
			})
			return
		}
		// The form shows whole, as its button needs its fields.
		s.found(c, locales.AccountSettingsConnectLastfm(), locales.AccountSettingsLastfmForm(), func() {
			s.steps(c,
				func() {
					ui.Text(c, locales.AccountSettingsLastfmStep1A())
					s.linkOf(c, "Last.fm", "https://www.last.fm/api/account/create")
					ui.Text(c, locales.AccountSettingsLastfmStep1B())
				},
				func() { ui.Text(c, locales.AccountSettingsLastfmStep2()) },
				func() { ui.Text(c, locales.AccountSettingsLastfmStep3()) },
			)
			s.row(c, locales.AccountSettingsAPIKey(), "", func() {
				ui.TextInput(c.Key("lastfm-key"), &s.lastfmKey).Placeholder(locales.AccountSettingsCharsPlaceholder()).Width(260).Label(locales.AccountSettingsAPIKey())
			})
			s.row(c, locales.AccountSettingsSharedSecret(), "", func() {
				ui.TextInput(c.Key("lastfm-secret"), &s.lastfmSecret).Password().Placeholder(locales.AccountSettingsCharsPlaceholder()).Width(260).Label(locales.AccountSettingsSharedSecret())
			})
			busy := s.signingIn == "lastfm"
			label := locales.AccountSettingsConnectButton()
			if busy {
				label = locales.AccountSettingsWaitingLastfm()
			}
			ui.Row(c).Padding(12, 0).Children(func() {
				key, secret := strings.TrimSpace(s.lastfmKey), strings.TrimSpace(s.lastfmSecret)
				if s.button(c, "lastfm-in", label, true).Disabled(busy || key == "" || secret == "").Clicked() {
					s.notes = append(s.notes, notice(locales.AccountSettingsApproveLastfm()))
					s.account("lastfm", locales.AccountSettingsConnectedLastfm(), func(ctx context.Context) error {
						_, err := s.svc.Accounts.LastfmLogin(ctx, key, secret)
						return err
					})
				}
			})
			s.hint(c, locales.AccountSettingsLastfmHint())
		})
	})
}

// steps lists what the user does, in order: paragraphs, which may hold
// links.
func (s *shell) steps(c *ui.Context, steps ...func()) {
	k := &s.lk
	ui.Column(c).Gap(8).Padding(12, 0).Children(func() {
		for i, step := range steps {
			ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
				ui.Text(c, fmt.Sprintf("%d.", i+1)).FontSize(k.bodySize()).TextColor(k.muted).Width(20).Shrink(0)
				ui.RichText(c).FontSize(k.bodySize()).Grow(1).MinWidth(0).Children(step)
			})
		}
	})
}

// account signs in or out beside the main thread, while kind, if any, says
// what waits; then it tells how it went, and the pages ask again for what
// they show, which the accounts change.
func (s *shell) account(kind, done string, do func(ctx context.Context) error) {
	s.signingIn = kind
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		err := do(ctx)
		s.update(func() {
			if s.signingIn == kind {
				s.signingIn = ""
			}
			if err != nil {
				s.notes = append(s.notes, noticeOf(err))
				s.forget("accounts")
				return
			}
			if done != "" {
				s.notes = append(s.notes, notice(done))
			}
			if kind == "lastfm" {
				s.lastfmSecret = ""
			}
			clear(s.loads)
		})
	}()
}
