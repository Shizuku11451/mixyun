package main

// What the user chose the window shows: the music sources on, and the
// sidebar's pages.

import (
	"slices"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/internal/settings"
	"github.com/fzn0x/mixmusic/locales"
)

func init() {
	addSettings("sources", 10, (*shell).sourceSettings)
	addSettings("appearance", 50, (*shell).sidebarSettings)
}

// sourceLabel names the music source of key, of settings.MusicSources.
func sourceLabel(key string) string {
	switch key {
	case settings.SourceNetEase:
		return "网易云音乐"
	case settings.SourceSpotify:
		return "Spotify"
	case settings.SourceYouTubeMusic:
		return "YouTube Music"
	case settings.SourceYouTube:
		return locales.SourcesYouTube()
	case settings.SourceFiles:
		return locales.SourcesFiles()
	}
	return locales.LibraryPodcasts()
}

// sourceOn tells whether the music source of the service src is on; what
// is no service's, as the user's own, always is.
func (s *shell) sourceOn(src media.Source) bool {
	switch src {
	case media.SourceSpotify:
		return s.set.SourceOn(settings.SourceSpotify)
	case media.SourceNetEase:
		return s.set.SourceOn(settings.SourceNetEase)
	case media.SourceYouTube:
		return s.set.SourceOn(settings.SourceYouTubeMusic)
	case media.SourceFile:
		return s.set.SourceOn(settings.SourceFiles)
	}
	return true
}

// sourceSettings turn the music sources on and off, in Settings › Music
// sources and in the guide of the first launch. The last one on can't go.
func (s *shell) sourceSettings(c *ui.Context) {
	s.group(c, locales.SourcesGroup(), func() {
		s.hint(c, locales.SourcesAbout())
		for _, key := range settings.MusicSources {
			on := s.set.SourceOn(key)
			last := on && len(s.set.SourcesOff) == len(settings.MusicSources)-1
			// Spotify's songs play matched on NetEase Cloud Music and
			// YouTube Music, which matching still searches while they're off.
			about := ""
			if !on && s.set.SourceOn(settings.SourceSpotify) && (key == settings.SourceNetEase || key == settings.SourceYouTubeMusic) {
				about = locales.SourcesMatchNote()
			}
			name := sourceLabel(key)
			s.row(c, name, about, func() {
				if s.switchOf(c.Key("source-"+key), &on).Label(name).Disabled(last).Changed() {
					off := slices.DeleteFunc(slices.Clone(s.set.SourcesOff), func(k string) bool { return k == key })
					if !on {
						off = append(off, key)
					}
					s.set.SourcesOff = off
					s.save()
				}
			}, "source", "service")
		}
		// shortcut: the sources keep a fixed order, in All and for the matcher; a drag order once users ask which comes first.
		s.resetRow(c, "sources-reset", locales.SourcesResetAbout(), len(s.set.SourcesOff) > 0, func() { s.set.SourcesOff = []string{} })
	})
}

// railPages are the pages of nav the sidebar shows: not those the user
// hid, nor Explore, which is YouTube Music's alone, while it's off.
func (s *shell) railPages() []navItem {
	var out []navItem
	for _, n := range nav {
		if slices.Contains(s.set.SidebarHidden, n.id) || n.id == "explore" && !s.set.SourceOn(settings.SourceYouTubeMusic) {
			continue
		}
		out = append(out, n)
	}
	return out
}

// sidebarSettings show and hide the sidebar's pages that may go, in
// Settings › Appearance.
func (s *shell) sidebarSettings(c *ui.Context) {
	s.group(c, locales.SidebarGroup(), func() {
		s.hint(c, locales.SidebarAbout())
		for _, n := range nav {
			if !slices.Contains(settings.SidebarPages, n.id) {
				continue
			}
			shown := !slices.Contains(s.set.SidebarHidden, n.id)
			name := n.label()
			s.row(c, name, "", func() {
				if s.switchOf(c.Key("sidebar-"+n.id), &shown).Label(name).Changed() {
					hidden := slices.DeleteFunc(slices.Clone(s.set.SidebarHidden), func(id string) bool { return id == n.id })
					if !shown {
						hidden = append(hidden, n.id)
					}
					s.set.SidebarHidden = hidden
					s.save()
				}
			}, "sidebar", "rail", "page")
		}
		// shortcut: the pages show and hide but keep their order; a drag order with upstream's fzn0x/mixmusic#588.
		s.resetRow(c, "sidebar-reset", locales.SidebarResetAbout(), len(s.set.SidebarHidden) > 0, func() { s.set.SidebarHidden = []string{} })
	})
}

// resetRow puts a part of the settings back as a new install has it, by
// reset, once changed tells it isn't.
func (s *shell) resetRow(c *ui.Context, key, about string, changed bool, reset func()) {
	s.row(c, locales.CustomizeReset(), about, func() {
		if s.button(c, key, locales.CustomizeResetButton(), false).Disabled(!changed).Clicked() {
			reset()
			s.save()
		}
	}, "default")
}
