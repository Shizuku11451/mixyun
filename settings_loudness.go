package main

import (
	"slices"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/settings"
	"github.com/fzn0x/mixmusic/locales"
)

func init() { addSettings(audioRows, 20, (*shell).loudnessRow) }

// loudnessRow turns loud tracks down to one level, in Settings › Audio
// › Sound.
func (s *shell) loudnessRow(c *ui.Context) {
	even := locales.LoudnessEvenDesc()
	if s.set.SpotifyPlayback == settings.SpotifyPlaybackConnect || s.set.SpotifyPlayback == settings.SpotifyPlaybackHere && spotifyPlaysHere {
		even = locales.LoudnessEvenDescSpotify()
	}
	s.row(c, locales.LoudnessEven(), even, func() {
		if s.switchOf(c.Key("loudness"), &s.set.Loudness).Label(locales.LoudnessEven()).Changed() {
			s.save()
		}
	})
}

func init() { addSettings(audioRows, 30, (*shell).levellingRow) }

// levellingRow squeezes loud and quiet moments closer together, in
// Settings › Audio › Output.
func (s *shell) levellingRow(c *ui.Context) {
	s.row(c, locales.LoudnessLevelling(), locales.LoudnessLevellingDesc(), func() {
		if s.switchOf(c.Key("normalize"), &s.set.Normalize).Label(locales.LoudnessLevelling()).Changed() {
			s.save()
		}
	}, "normalize loudness")
}

func init() { addSettings(audioRows, 25, (*shell).replayGainRow) }

// replayGainRow plays the music on this computer by its files' ReplayGain
// tags, in Settings › Audio › Output (#23).
func (s *shell) replayGainRow(c *ui.Context) {
	modes := []settings.ReplayGain{settings.ReplayGainOff, settings.ReplayGainTrack, settings.ReplayGainAlbum}
	s.row(c, locales.ReplayGain(), locales.ReplayGainDesc(), func() {
		chosen := slices.Index(modes, s.set.ReplayGain)
		if s.segmented(c, "replay-gain", locales.ReplayGain(), &chosen, locales.ReplayGainOff(), locales.ReplayGainTrack(), locales.ReplayGainAlbum()) {
			s.set.ReplayGain = modes[chosen]
			s.save()
		}
	}, "replaygain normalize volume album")
}
