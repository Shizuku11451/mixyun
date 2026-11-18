package main

import (
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/locales"
)

func init() { addSettings("content", 30, (*shell).skipSettings) }

// skipRows are the rows of Settings › Content › Songs to skip, which
// register from their features' files.
const skipRows = "content › Songs to skip"

// skipSettings are the songs that play passes over.
func (s *shell) skipSettings(c *ui.Context) {
	s.group(c, locales.ContentSkipGroup(), func() { s.settingsOf(c, skipRows) })
}

func init() { addSettings(skipRows, 10, (*shell).skipAIRow) }

// skipAIRow skips the tracks YouTube labels as made with AI, in Settings ›
// Content › Songs to skip.
func (s *shell) skipAIRow(c *ui.Context) {
	s.row(c, locales.SkipAITitle(), locales.SkipAIDesc(), func() {
		if s.switchOf(c.Key("skip-ai"), &s.set.SkipAI).Label(locales.SkipAITitle()).Changed() {
			s.save()
		}
	})
}
