package main

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/locales"
)

// welcomeGuide is the guide of the first launch: open while it shows, at
// step of welcomeSteps.
type welcomeGuide struct {
	open bool
	step int
}

// welcomeSteps are the steps of the guide of the first launch: each builds
// the settings' own groups, whose defaults hold until the user changes
// them, so that Next straight through is a working setup. The steps change
// at once, without sliding, for the desktop that asks for less motion too.
var welcomeSteps = []func(*shell, *ui.Context){
	(*shell).languageSettings,
	func(s *shell, c *ui.Context) {
		s.themeSettings(c)
		s.layoutSettings(c)
	},
	(*shell).accountSettings,
	func(s *shell, c *ui.Context) {
		s.group(c, locales.LyricsSettingsGroup(), func() { s.translateToRow(c) })
	},
	(*shell).startSettings,
}

// welcomeFirstRun opens the guide of the first launch in a new install,
// which had no settings.json as MixMusic started, until the user finished
// or skipped it. Installs that had one never see it.
func (s *shell) welcomeFirstRun() {
	s.welcome.open = s.svc.Preferences.FirstRun() && !s.set.FirstRunDone
}

// openWelcome opens the guide of the first launch again, at its first step.
func (s *shell) openWelcome() { s.welcome = welcomeGuide{open: true} }

// welcomeDialog shows the guide of the first launch while it's open: a
// step, with Skip, Back and Next. Finishing it, Skip, Escape or a click
// outside it close it, keeping what the user chose, and save that it was
// done.
func (s *shell) welcomeDialog(c *ui.Context) {
	g := &s.welcome
	if !g.open {
		return
	}
	last := g.step == len(welcomeSteps)-1
	// A key of each step's own starts its body at the top.
	key := "welcome-" + strconv.Itoa(g.step)
	s.dialog(c, &g.open, dialogOptions{key: key, title: locales.WelcomeTitle(), width: 640}, func() {
		ui.Text(c, locales.WelcomeStep(g.step+1, len(welcomeSteps))).FontSize(s.lk.smallSize()).TextColor(s.lk.muted)
		welcomeSteps[g.step](s, c)
	}, func() {
		if s.dialogButton(c, "welcome-skip", locales.WelcomeSkip(), false).Clicked() {
			g.open = false
		}
		if g.step > 0 && s.dialogButton(c, "welcome-back", locales.WelcomeBack(), false).Clicked() {
			g.step--
		}
		next := locales.WelcomeNext()
		if last {
			next = locales.WelcomeDone()
		}
		if s.dialogButton(c, "welcome-next", next, true).AutoFocus().Clicked() {
			if last {
				g.open = false
			} else {
				g.step++
			}
		}
	})
	if !g.open {
		*g = welcomeGuide{}
		if !s.set.FirstRunDone {
			s.set.FirstRunDone = true
			s.save()
		}
	}
}

func init() { addSettings("general", 40, (*shell).welcomeSettings) }

// welcomeSettings open the guide of the first launch again.
func (s *shell) welcomeSettings(c *ui.Context) {
	s.group(c, locales.WelcomeGuide(), func() {
		s.row(c, locales.WelcomeGuide(), locales.WelcomeGuideAbout(), func() {
			if s.button(c, "welcome-open", locales.WelcomeGuideOpen(), false).Clicked() {
				s.openWelcome()
			}
		}, "first run", "onboarding", "setup")
	})
}
