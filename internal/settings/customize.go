package settings

import "slices"

// The music sources the user may turn off, by their keys in SourcesOff.
const (
	SourceNetEase      = "netease"
	SourceSpotify      = "spotify"
	SourceYouTubeMusic = "youtubeMusic"
	// SourceYouTube is YouTube's own videos, which YouTube Music doesn't
	// list.
	SourceYouTube  = "youtube"
	SourceFiles    = "files"
	SourcePodcasts = "podcasts"
)

// MusicSources are the sources the user may turn off, in the order
// Settings › Music sources lists them.
var MusicSources = []string{SourceNetEase, SourceSpotify, SourceYouTubeMusic, SourceYouTube, SourceFiles, SourcePodcasts}

// SidebarPages are the sidebar's pages the user may hide, by their ids;
// Home, Search, Library and Settings always show.
var SidebarPages = []string{"explore", "roam", "stats", "marketplace"}

// Customize is what the user chose the window shows: the music sources,
// Home's pivot and the sidebar's pages.
type Customize struct {
	// SourcesOff are the music sources the user turned off, of
	// MusicSources: they leave Home, Search, the library and Accounts,
	// though their accounts stay signed in and matching still uses them.
	// One source at least stays on.
	SourcesOff []string `json:"sourcesOff"`
	// HomePivot is the pivot Home shows, as the user chose it last: ""
	// for Everything, or the source it keeps, as "local" or "netease".
	HomePivot string `json:"homePivot"`
	// SidebarHidden are the pages of SidebarPages the user took off the
	// sidebar.
	SidebarHidden []string `json:"sidebarHidden"`
}

// SourceOn tells whether the music source of key, of MusicSources, is on.
func (c Customize) SourceOn(key string) bool { return !slices.Contains(c.SourcesOff, key) }

// defaultCustomize is the window of a new install: every source on, and
// every page in the sidebar.
func defaultCustomize() Customize {
	return Customize{SourcesOff: []string{}, SidebarHidden: []string{}}
}

// clean is c with the sources and the pages it names known, each once,
// and the first source on when it turned every one off.
func (c Customize) clean() Customize {
	c.SourcesOff = known(c.SourcesOff, MusicSources)
	if len(c.SourcesOff) == len(MusicSources) {
		c.SourcesOff = slices.DeleteFunc(c.SourcesOff, func(k string) bool { return k == MusicSources[0] })
	}
	c.SidebarHidden = known(c.SidebarHidden, SidebarPages)
	switch c.HomePivot {
	case "", "local", "spotify", "netease", "youtube":
	default:
		c.HomePivot = ""
	}
	return c
}

// known are the keys of keys that of names, each once, in a list of its
// own.
func known(keys, of []string) []string {
	out := []string{}
	for _, k := range keys {
		if slices.Contains(of, k) && !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}
