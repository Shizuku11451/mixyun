package main

import (
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/locales"
)

func init() { addSettings(audioRows, 40, (*shell).surroundRow) }

// surroundRow plays the 5.1 surround mix of the tracks that have one, in
// Settings › Audio › Output.
func (s *shell) surroundRow(c *ui.Context) {
	s.row(c, locales.SurroundTitle(), surroundDescription(), func() {
		if s.switchOf(c.Key("surround"), &s.set.Surround).Label(locales.SurroundTitle()).Changed() {
			s.save()
		}
	})
}

// surroundDescription tells what 5.1 surround does, in Settings › Audio.
func surroundDescription() string { return locales.SurroundDescription() }
