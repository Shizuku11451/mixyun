package main

import (
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/locales"
)

// The row is not shown: it serves YouTube Music's tracks alone, which
// NetEase Cloud Music took the place of in the interface.

// skipAIRow skips the tracks YouTube labels as made with AI, in Settings ›
// Playback › Sound.
func (s *shell) skipAIRow(c *ui.Context) {
	s.row(c, locales.SkipAITitle(), locales.SkipAIDesc(), func() {
		if s.switchOf(c.Key("skip-ai"), &s.set.SkipAI).Label(locales.SkipAITitle()).Changed() {
			s.save()
		}
	})
}
