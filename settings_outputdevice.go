package main

import (
	"runtime"
	"slices"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/locales"
)

func init() { addSettings("audio", 10, (*shell).outputSettings) }

// audioRows are the rows of Settings › Audio › Output, which register from
// their features' files.
const audioRows = "audio › Output"

// outputSettings are where and how the sound comes out: the device, its
// loudness and surround.
func (s *shell) outputSettings(c *ui.Context) {
	s.group(c, locales.AudioOutputGroup(), func() { s.settingsOf(c, audioRows) })
}

func init() { addSettings(audioRows, 10, (*shell).outputDeviceRow) }

// outputDeviceRow chooses the device MixMusic's own player plays on, in
// Settings › Audio › Output: the system's default, or a device by name.
func (s *shell) outputDeviceRow(c *ui.Context) {
	description := outputDescription()
	if s.set.OutputDevice.ID != "" && s.svc.Player.OutputFallback() {
		description = locales.OutputDescriptionGone(s.set.OutputDevice.Name)
	}
	s.row(c, locales.OutputDevice(), description, func() {
		devices := s.outputDevices()
		choices := outputChoices(devices.value, devices.done && devices.err == nil, s.set.OutputDevice)
		titles := make([]string, len(choices))
		chosen := ""
		for i, o := range choices {
			titles[i] = o.title()
			if o.Chosen {
				chosen = titles[i]
			}
		}
		if s.selectOf(c.Key("output-device"), &chosen, titles).Width(260).Label(locales.OutputDevice()).Changed() {
			if i := slices.Index(titles, chosen); i >= 0 {
				s.chooseOutput(choices[i].Device)
			}
		}
		if s.iconButton(c, "output-again", "refresh", locales.OutputLookAgain(), false, 16).Clicked() {
			s.refetch(outputDevicesKey)
		}
	}, "speakers", "headphones", "sound card", "dac", "audio device")
}

// outputDescription tells what the output device does, in Settings ›
// Playback.
func outputDescription() string { return locales.OutputDescription() }

func init() { addSettings(audioRows, 11, (*shell).nativeRateRow) }

// nativeRateRow has the device chosen switch to the rate of the FLAC that
// plays, in Settings › Audio › Output: on macOS, with a device chosen.
func (s *shell) nativeRateRow(c *ui.Context) {
	if runtime.GOOS != "darwin" || s.set.OutputDevice.ID == "" {
		return
	}
	s.row(c, locales.OutputNativeRateTitle(), locales.OutputNativeRateDescription(), func() {
		if s.switchOf(c.Key("native-rate"), &s.set.OutputDevice.NativeRate).Label(locales.OutputNativeRateTitle()).Changed() {
			s.save()
		}
	}, "sample rate", "44.1", "96", "hi-res", "dac")
}
