package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/addons"
	"github.com/fzn0x/mixmusic/locales"
)

// settingsTabs are the settings' sections, by their paths' last part.
var settingsTabs = []string{"general", "appearance", "playback", "audio", "lyrics", "content", "accounts", "storage", "desktop", "plugins", "shortcuts", "about"}

// settingsTabName names a section of settingsTabs, in the language shown.
func settingsTabName(tab string) string {
	switch tab {
	case "general":
		return locales.SettingsTabGeneral()
	case "appearance":
		return locales.SettingsTabAppearance()
	case "playback":
		return locales.SettingsTabPlayback()
	case "audio":
		return locales.SettingsTabAudio()
	case "lyrics":
		return locales.SettingsTabLyrics()
	case "content":
		return locales.SettingsTabContent()
	case "accounts":
		return locales.SettingsTabAccounts()
	case "storage":
		return locales.SettingsTabStorage()
	case "plugins":
		return locales.SettingsTabPlugins()
	case "desktop":
		return locales.SettingsTabDesktop()
	case "shortcuts":
		return locales.SettingsTabShortcuts()
	case "about":
		return locales.SettingsTabAbout()
	}
	return tab
}

// settingsPage shows the settings, by section, or, searched for q, what
// every section holds of it.
func (s *shell) settingsPage(c *ui.Context, tab, q string) {
	f := &s.find
	// The page the router shows takes the search its location holds, as a
	// link, Back or the palette set it. The one it leaves, which shows
	// beside it a moment, shows its own.
	current := s.pagePath == s.router.Path() && q == s.router.Query("q")
	if current && q != f.route {
		f.query, f.route, f.typedAt = q, q, c.Now()
	}
	selected := max(slices.Index(settingsTabs, tab), 0)
	s.pageScroll(c, func() {
		f.begin(q, current)
		defer f.end()
		// While the user searches, no section is chosen; choosing one
		// leaves the search.
		pick := selected
		if f.on() {
			pick = -1
		}
		before := pick
		names := make([]string, len(settingsTabs))
		for i, t := range settingsTabs {
			names[i] = settingsTabName(t)
		}
		// The sections list at the page's left where it has the room, and
		// show as pivots over it where it hasn't.
		side := s.paneWidth(c) >= settingsSideMin
		if side {
			s.pageHeader(c, locales.SettingsTitle(), nil, nil)
		} else {
			s.pageHeader(c, locales.SettingsTitle(), names, &pick)
		}
		s.findBox(c, tab, q, current)
		body := func() {
			ui.Column(c).Grow(1).MinWidth(0).MaxWidth(880).Children(func() {
				s.settingsFileProblem(c)
				if !f.on() {
					s.sectionSettings(c, settingsTabs[selected])
					if current {
						f.reveal = "" // a group the section doesn't show
					}
					return
				}
				for _, t := range settingsTabs {
					f.section = t
					s.sectionSettings(c, t)
				}
				f.section = ""
				if f.found == 0 {
					s.emptyLine(c, locales.SettingsNoMatch(strings.TrimSpace(q)))
				}
			})
		}
		if side {
			ui.Row(c).Gap(40).AlignItems(ui.Start).Children(func() {
				s.settingsSections(c, names, &pick)
				body()
			})
		} else {
			body()
		}
		if pick != before {
			s.router.Replace(settingsPath(settingsTabs[pick], ""))
		}
		if current {
			s.findTell(c)
		}
	})
}

// settingsSideMin is the narrowest pane the settings' sections list at
// the page's left in.
const settingsSideMin = 760

// settingsSections lists the settings' sections, names, at the page's
// left: the chosen one, *selected, filled.
func (s *shell) settingsSections(c *ui.Context, names []string, selected *int) {
	k := &s.lk
	ui.Column(c).Width(184).Shrink(0).Gap(2).Role(ui.RoleTabList).Label(locales.SettingsTitle()).Children(func() {
		for i, name := range names {
			name := k.sentence(name)
			b := s.buttonBase(c.Key("section-" + settingsTabs[i])).Role(ui.RoleTab).Label(name).AccessSelected(i == *selected).
				Height(36).PaddingX(12).Radius(k.radiusSm).Cursor(ui.CursorPointer)
			fill, ink := ui.Color{}, k.text.Alpha(0.75)
			switch {
			case i == *selected:
				fill, ink = k.selected, k.selectedText
			case b.Pressed():
				fill, ink = k.surface3, k.text
			case b.Hovered():
				fill, ink = k.surface2, k.text
			}
			b.Background(fill)
			if b.Clicked() {
				*selected = i
			}
			b.Children(func() {
				ui.Text(c, name).FontSize(k.bodySize()).TextColor(ink).SingleLine().MinWidth(0)
			})
		}
	})
}

// advancedOf is the place of the groups of section that few change, which
// fold under Advanced at its foot.
func advancedOf(section string) string { return section + " › advanced" }

// sectionSettings builds a section of the settings: its groups, then those
// under Advanced, folded unless the user opened the fold. The search looks
// through the folded groups too.
func (s *shell) sectionSettings(c *ui.Context, section string) {
	s.settingsOf(c, section)
	place := advancedOf(section)
	if len(settingsParts[place]) == 0 {
		return
	}
	f := &s.find
	if f.on() {
		f.advanced = true
		s.settingsOf(c, place)
		f.advanced = false
		return
	}
	k := &s.lk
	open := s.set.AdvancedSettings
	label := locales.SettingsAdvanced()
	b := s.buttonBase(c.Key("advanced-"+section)).Label(label).Expanded(open).Gap(8).Height(36).Margin(0, 0, 16).
		AlignSelf(ui.Start).Cursor(ui.CursorPointer)
	ink := k.muted
	if b.Hovered() {
		ink = k.text
	}
	if b.Clicked() {
		s.set.AdvancedSettings = !open
		s.save()
	}
	b.Children(func() {
		chevron := "chevronRight"
		if open {
			chevron = "chevronDown"
		}
		ui.Icon(c, icon(chevron)).FontSize(16).TextColor(ink)
		ui.Text(c, k.sentence(label)).FontSize(k.bodySize()).TextColor(ink)
	})
	if open {
		s.settingsOf(c, place)
	}
}

// A settingsPart is a part of a place of the settings page, which builds
// from its feature's file: a group of a section, or a row of a group whose
// rows register.
type settingsPart struct {
	order int
	build func(*shell, *ui.Context)
}

// settingsParts are the parts of each place of the settings page, in their
// order, as addSettings registered them.
var settingsParts = map[string][]settingsPart{}

// addSettings registers build as a part of place on the settings page, at
// order among place's parts. A place is a section, by its tab's name in
// settingsTabs, as "playback", or a group whose rows register, which names
// its section first, as soundRows: "playback › Sound". Each part registers
// from its feature's own file, in an init function, so that a new setting
// edits only that file: orders go by tens, for a new part to slot in
// between, and two parts of a place can't share one.
func addSettings(place string, order int, build func(*shell, *ui.Context)) {
	if section, _, _ := strings.Cut(place, " › "); !slices.Contains(settingsTabs, section) {
		panic("settings: " + place + " is in no section")
	}
	settingsParts[place] = withPart(settingsParts[place], place, order, build)
}

// withPart is parts, the parts of place in their order, with build at
// order among them.
func withPart(parts []settingsPart, place string, order int, build func(*shell, *ui.Context)) []settingsPart {
	i, taken := slices.BinarySearchFunc(parts, order, func(p settingsPart, order int) int { return cmp.Compare(p.order, order) })
	if taken {
		panic(fmt.Sprintf("settings: two parts of %s at %d", place, order))
	}
	return slices.Insert(parts, i, settingsPart{order, build})
}

// settingsOf builds the parts of place, in their order: a section of the
// settings, or the rows of a group.
func (s *shell) settingsOf(c *ui.Context, place string) {
	for _, p := range settingsParts[place] {
		p.build(s, c)
	}
}

// save keeps what the controls changed of the settings: only that, so
// that an edit of settings.json that applied since refresh read them, which
// the window doesn't show until its refresh runs, stays.
func (s *shell) save() {
	next := s.set.Clean()
	if _, err := s.svc.Preferences.Change(s.base, next); err != nil {
		s.notes = append(s.notes, noticeOf(err))
		return
	}
	s.base = next
}

// group titles a set of rows.
func (s *shell) group(c *ui.Context, title string, rows func()) { s.groupWith(c, title, nil, rows) }

// groupWith titles a set of rows, with links to what relates to them.
// While the user searches the settings, it shows what the search finds of
// them (findGroup).
func (s *shell) groupWith(c *ui.Context, title string, links, rows func()) {
	k := &s.lk
	f := &s.find
	if f.on() {
		s.findGroup(c, title, links, rows)
		return
	}
	g := ui.Column(c).Margin(0, 0, k.sections())
	// The group a search's link showed in its section comes into view.
	if f.reveal != "" && f.reveal == title && f.current {
		f.reveal = ""
		g.ScrollIntoView()
	}
	g.Children(func() {
		s.sectionHead(c, title, links)
		group := ui.Column(c).Dividers(1, k.border)
		if k.cards {
			group.Background(k.surface).Radius(k.radiusSm+k.cardPad()).Padding(4, 2*k.cardPad())
		}
		group.Children(rows)
	})
}

// row is a setting: its title and description, and its control. The
// search of the settings finds it by them, and by words.
func (s *shell) row(c *ui.Context, title, description string, control func(), words ...string) {
	if !s.find.on() {
		s.rowOf(c, title, description, control)
		return
	}
	if s.findable(title, description, words...) {
		s.show(c, func() { s.rowOf(c, title, description, control) })
	}
}

// rowOf builds a setting's row.
func (s *shell) rowOf(c *ui.Context, title, description string, control func()) {
	k := &s.lk
	ui.Row(c).Gap(24).Padding(14, 0).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, title).FontSize(k.bodySize())
			if description != "" {
				ui.Text(c, description).FontSize(k.smallSize()).TextColor(k.muted)
			}
		})
		ui.Row(c).Shrink(0).Gap(8).Children(control)
	})
}

// draft is the text of a setting's input, until the user is done with it.
type draft struct {
	text    string
	focused bool
}

// settingInput is a text input of a setting that holds saved: the user's
// text is a draft until they press Enter or leave the input, when commit
// takes it, trimmed. Until they edit it, it shows the setting.
func (s *shell) settingInput(c *ui.Context, key, saved string, commit func(string)) (ui.Element, *draft) {
	d := s.drafts[key]
	if d == nil {
		d = &draft{text: saved}
		s.drafts[key] = d
	}
	in := ui.TextInput(c.Key(key), &d.text)
	focused := in.Focused()
	switch {
	case in.Submitted() || (d.focused && !focused):
		if v := strings.TrimSpace(d.text); v != saved {
			commit(v)
		}
	case !focused:
		d.text = saved
	}
	d.focused = focused
	return in, d
}

// rowWith is a row whose description describe builds, with links say.
// The search of the settings finds it by its title, and by words, which
// say what describe does.
func (s *shell) rowWith(c *ui.Context, title string, describe, control func(), words ...string) {
	if !s.findable(title, "", words...) {
		return
	}
	k := &s.lk
	s.show(c, func() {
		ui.Row(c).Gap(24).Padding(14, 0).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, title).FontSize(k.bodySize())
				ui.RichText(c).FontSize(k.smallSize()).TextColor(k.muted).Children(describe)
			})
			ui.Row(c).Shrink(0).Gap(8).Children(control)
		})
	})
}

// hint is a muted paragraph among a group's rows.
func (s *shell) hint(c *ui.Context, text string) {
	s.found(c, "", text, func() {
		ui.Text(c, text).FontSize(s.lk.smallSize()).TextColor(s.lk.muted).Padding(12, 0).MaxWidth(720)
	})
}

// problems tells of the add-ons of a kind that didn't load.
func (s *shell) problems(c *ui.Context, kind addons.Kind) {
	what := "theme"
	if kind == addons.KindPlugin {
		what = "plugin"
	}
	for _, p := range s.inv.Problems {
		if p.Kind == kind {
			text := locales.SettingsAddonProblem(what, p.Folder, p.Error)
			s.found(c, "", text, func() {
				ui.Text(c, text).FontSize(s.lk.smallSize()).TextColor(s.lk.danger).Padding(8, 0)
			})
		}
	}
}

// openFolder opens the folder of the add-ons of a kind.
func (s *shell) openFolder(kind addons.Kind) {
	if err := s.svc.Addons.OpenFolder(kind); err != nil {
		s.notes = append(s.notes, noticeOf(err))
	}
}
