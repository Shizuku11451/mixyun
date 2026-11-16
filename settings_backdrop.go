package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/locales"
)

func init() { addSettings(colorTypeRows, 80, (*shell).coverBackdropRows) }

// coverBackdropRows turn the cover's backdrop of the now playing page on
// and off, and set its veil, in Settings › Appearance › Colour and type.
func (s *shell) coverBackdropRows(c *ui.Context) {
	s.row(c, locales.CoverBackdropTitle(), locales.CoverBackdropDescription(), func() {
		if s.switchOf(c.Key("cover-backdrop"), &s.set.CoverBackdrop).Label(locales.CoverBackdropTitle()).Changed() {
			s.save()
		}
	}, "blur", "background", "apple music")
	if !s.set.CoverBackdrop {
		return
	}
	veil := fmt.Sprintf("%d%% · %s", int(s.set.CoverBackdropVeil*100+0.5), locales.CoverBackdropVeilDescription())
	s.row(c, locales.CoverBackdropVeilTitle(), veil, func() {
		if ui.StepSlider(c.Key("cover-backdrop-veil"), &s.set.CoverBackdropVeil, 0, 0.9, 0.05).Width(200).Label(locales.CoverBackdropVeilTitle()).Changed() {
			s.save()
		}
	}, "opacity", "transparency", "blur")
}
