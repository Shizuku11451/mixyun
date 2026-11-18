package main

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/internal/settings"
	"github.com/fzn0x/mixmusic/locales"
	"github.com/fzn0x/mixmusic/lyrics"
)

// translateLanguages are the languages Settings › Playback offers to
// translate lyrics to, by their codes of ISO 639-1.
var translateLanguages = []struct {
	code string
	name func() string
}{
	{"af", locales.LyricsLangAfrikaans}, {"ar", locales.LyricsLangArabic}, {"bg", locales.LyricsLangBulgarian},
	{"bn", locales.LyricsLangBengali}, {"ca", locales.LyricsLangCatalan}, {"zh", locales.LyricsLangChineseSimplified},
	{"zh-Hant", locales.LyricsLangChineseTraditional}, {"hr", locales.LyricsLangCroatian},
	{"cs", locales.LyricsLangCzech}, {"da", locales.LyricsLangDanish}, {"nl", locales.LyricsLangDutch},
	{"en", locales.LyricsLangEnglish}, {"et", locales.LyricsLangEstonian}, {"tl", locales.LyricsLangFilipino},
	{"fi", locales.LyricsLangFinnish}, {"fr", locales.LyricsLangFrench}, {"de", locales.LyricsLangGerman},
	{"el", locales.LyricsLangGreek}, {"he", locales.LyricsLangHebrew}, {"hi", locales.LyricsLangHindi},
	{"hu", locales.LyricsLangHungarian}, {"id", locales.LyricsLangIndonesian}, {"it", locales.LyricsLangItalian},
	{"ja", locales.LyricsLangJapanese}, {"ko", locales.LyricsLangKorean}, {"lv", locales.LyricsLangLatvian},
	{"lt", locales.LyricsLangLithuanian}, {"ms", locales.LyricsLangMalay}, {"no", locales.LyricsLangNorwegian},
	{"fa", locales.LyricsLangPersian}, {"pl", locales.LyricsLangPolish}, {"pt", locales.LyricsLangPortuguese},
	{"ro", locales.LyricsLangRomanian}, {"ru", locales.LyricsLangRussian}, {"sr", locales.LyricsLangSerbian},
	{"sk", locales.LyricsLangSlovak}, {"sl", locales.LyricsLangSlovenian}, {"es", locales.LyricsLangSpanish},
	{"sw", locales.LyricsLangSwahili}, {"sv", locales.LyricsLangSwedish}, {"ta", locales.LyricsLangTamil},
	{"te", locales.LyricsLangTelugu}, {"th", locales.LyricsLangThai}, {"tr", locales.LyricsLangTurkish},
	{"uk", locales.LyricsLangUkrainian}, {"ur", locales.LyricsLangUrdu}, {"vi", locales.LyricsLangVietnamese},
}

// romanizeNames are the names of the switch that shows lyrics in Latin
// letters, and of its ways, for Japanese (romaji) and for the other languages.
func romanizeNames(romaji bool) [3]string {
	if romaji {
		return [3]string{locales.LyricsRomaji(), locales.LyricsRomajiUnder(), locales.LyricsRomajiOnly()}
	}
	return [3]string{locales.LyricsRomanize(), locales.LyricsRomanizeUnder(), locales.LyricsRomanizeOnly()}
}

// translateTarget is the language lyrics translate to: the one chosen in
// Settings › Playback, else the system's.
func (s *shell) translateTarget() string {
	if s.set.LyricsTranslateTo != "" {
		return s.set.LyricsTranslateTo
	}
	return s.systemLanguage()
}

// systemLanguage is the system's language, asked for once.
func (s *shell) systemLanguage() string {
	ls := &s.lyricsView
	if ls.system == "" {
		ls.system = s.svc.System.Language()
	}
	return ls.system
}

// romanizedFirst are the languages whose lyrics Unison romanizes: Korean,
// Chinese, Russian and Arabic lines came back romanized on 2026-11-02, as
// Google's translation of lyrics romanizes them, and Japanese, Thai,
// Hebrew, Hindi, Greek and Ukrainian ones didn't. MixMusic romanizes all
// of these itself (lyricsromaji.go, lyricsromanize.go): Unison is asked
// only for Arabic's, which are better (unisonFirst), and, for lyrics in
// a script MixMusic doesn't romanize, its switch shows once a translation
// of them brought some Latin letters.
var romanizedFirst = map[string]bool{"ar": true}

// translated is the translation of lyrics, and whether any line of it came
// in Latin letters, found beside the main thread.
type translated struct {
	lyrics.Translation
	romanized bool
}

// lyricsExtra is what the lines of lyrics show besides their words, as
// the user chose for their language: their translation, and their words
// in Latin letters, under them or in their place.
type lyricsExtra struct {
	translate bool
	romanize  string
	// tr is the translation of the lyrics, once it came, which holds their
	// words in Latin letters too; nil until then, and while neither shows.
	tr *lyrics.Translation
	// ja is set for lyrics in Japanese, whose words in Latin letters are
	// their romaji, made on the computer: nil until made.
	ja     bool
	romaji []string
	// local is the language of lyrics romanized on the computer
	// (lyricsromanize.go), and roman their lines romanized: nil until made.
	local string
	roman *romanizedLines
}

// line returns what line i of the lyrics shows: its words, as written or
// in Latin letters, then those in Latin letters under it, and its
// translation, "" where there are none.
func (x lyricsExtra) line(i int, words string) (text, roman, meaning string) {
	text = words
	latin := ""
	switch {
	case x.ja:
		if x.romaji != nil {
			latin = x.romaji[i]
		}
	case x.local != "":
		// Unison's, where it is better, once it came.
		if x.tr != nil && unisonFirst[x.local] {
			latin = x.tr.Lines[i].Romanized
		}
		if latin == "" && x.roman != nil {
			latin = x.roman.lines[i]
		}
	case x.tr != nil:
		latin = x.tr.Lines[i].Romanized
	}
	switch {
	case latin == "":
	case x.romanize == settings.RomanizeOnly:
		text = latin
	case x.romanize == settings.RomanizeUnder:
		roman = latin
	}
	if x.translate && x.tr != nil {
		meaning = x.tr.Lines[i].Text
	}
	return text, roman, meaning
}

// lyricsChoices shows how lyrics show, above them: synced or as plain
// text, translated, and in Latin letters, each where it can; and, while
// their translation loads or when it fails, says so. It tells what their
// lines show besides their words.
func (s *shell) lyricsChoices(c *ui.Context, t media.Track, ly lyrics.Lyrics, where string) lyricsExtra {
	return s.lyricsChoicesWith(c, t, ly, where, lyricsChoiceOptions{})
}

// lyricsChoiceOptions say what lyricsChoicesWith shows. Its zero value
// shows what lyricsChoices does.
type lyricsChoiceOptions struct {
	// quiet tells what the lines show without drawing the switches and the
	// notes above them, as the full-screen lyrics (lyricsstage.go) have
	// none: they show what the user chose in the panel.
	quiet bool
}

// lyricsChoicesWith is lyricsChoices for the options o.
func (s *shell) lyricsChoicesWith(c *ui.Context, t media.Track, ly lyrics.Lyrics, where string, o lyricsChoiceOptions) lyricsExtra {
	lang, to := ly.Language, s.translateTarget()
	// Lyrics of a language MixMusic can't tell may be in another.
	canTranslate := lang != to
	x := lyricsExtra{translate: canTranslate && slices.Contains(s.set.LyricsTranslate, lang), ja: s.japanese(ly), local: s.romanizeLanguage(ly)}
	// Lyrics in another script show in Latin letters made on the computer,
	// which asks Unison for nothing: Japanese in romaji, and the others
	// as their scripts are romanized, but Arabic, as Unison has it once it
	// came. Lyrics in a script MixMusic doesn't romanize show as Unison
	// has them, with their translation, if it brought them so.
	romanize := ""
	switch {
	case x.ja:
		romanize = s.set.LyricsRomanize["ja"]
	case x.local != "":
		romanize = s.set.LyricsRomanize[x.local]
	case lyrics.Romanizable(lang):
		romanize = s.set.LyricsRomanize[lang]
	}
	var tr *loaded[translated]
	if x.translate || !x.ja && romanize != "" && (x.local == "" && romanizedFirst[lang] || unisonFirst[x.local] && romanizedFirst[x.local]) {
		tr = s.translation(t, ly, to, x.translate)
		if tr.done && tr.err == nil && len(tr.value.Lines) == len(ly.Lines) {
			x.tr = &tr.value.Translation
		}
	}
	canRomanize := x.ja || x.local != "" || lyrics.Romanizable(lang) && (romanizedFirst[lang] || x.translate && x.tr != nil && tr.value.romanized)
	if canRomanize {
		x.romanize = romanize
	}
	// The share link shows with the switches, whatever they offer.
	switch {
	case o.quiet:
	case where == "page":
		s.lyricsRail(c, t, ly, lang, canTranslate, canRomanize, &x)
	default:
		s.lyricsSwitches(c, ly, lang, where, canTranslate, canRomanize, &x, func() { s.lyricsShareStart(c, t, where) })
	}
	if x.ja && x.romanize != "" {
		x.romaji = s.lyricsRomaji(c, t, ly, where, o.quiet)
	}
	if x.local != "" && x.romanize != "" {
		x.roman = s.lyricsRomanized(t, ly, x.local)
	}
	if tr == nil || o.quiet {
		return x
	}
	waiting, failed := locales.LyricsRomanizationWaiting(), locales.LyricsRomanizationFailed()
	if x.translate {
		waiting, failed = locales.LyricsTranslationWaiting(), locales.LyricsTranslationFailed()
	}
	switch {
	case !tr.done && (x.translate || x.roman == nil):
		s.lyricsNote(c, waiting)
	case tr.err != nil && (x.translate || x.local == ""):
		// One quiet line; the lyrics show all the same.
		k := &s.lk
		ui.Row(c).Padding(0, 24, 8).Gap(12).AlignItems(ui.Center).Shrink(0).Children(func() {
			ui.Text(c, failed).FontSize(k.smallSize()).TextColor(k.muted)
			if s.link(c, "lyrics-translate-again-"+where, locales.WidgetsTryAgain()) {
				s.svc.Catalog.TranslateAgain()
				s.forget(s.translationKey(t, translationInto(ly, to, x.translate)))
			}
		})
	}
	return x
}

// translation fetches the translation of a track's lyrics into the
// language to, as translationInto has it.
func (s *shell) translation(t media.Track, ly lyrics.Lyrics, to string, translate bool) *loaded[translated] {
	to = translationInto(ly, to, translate)
	return fetch(s, s.translationKey(t, to), func(ctx context.Context) (translated, error) {
		tr, err := s.svc.Catalog.TranslateLyrics(ctx, t, ly, to)
		return translated{tr, slices.ContainsFunc(tr.Lines, func(l lyrics.TranslatedLine) bool { return l.Romanized != "" })}, err
	})
}

// translationInto is the language to ask a translation of lyrics into,
// for their translation into to: to, but, for their romanization alone,
// translate false, English when to is their own language, as Unison
// translates nothing into the language lyrics are in.
func translationInto(ly lyrics.Lyrics, to string, translate bool) string {
	if !translate && ly.Language == to {
		return "en"
	}
	return to
}

// translationKey is the key translation fetches the translation of a
// track's lyrics into the language into under.
func (s *shell) translationKey(t media.Track, into string) string {
	key := "lyrics-translation:" + t.ID + ":" + into
	// Another language model translates anew.
	if s.set.LLM() {
		key += ":" + s.set.LyricsLLMURL + " " + s.set.LyricsLLMModel
	}
	return key
}

// lyricsSwitches shows the choices above lyrics in the language lang:
// synced or as plain text, when they are synced; translated, when they
// can be; and in Latin letters, under each line or in its place, when
// they can be; and, last, share, which starts sharing lines. It changes x
// as the user chooses. The now playing page has them at the side instead
// (lyricsRail).
func (s *shell) lyricsSwitches(c *ui.Context, ly lyrics.Lyrics, lang, where string, canTranslate, canRomanize bool, x *lyricsExtra, share func()) {
	ui.Row(c).Padding(0, 24, 8).GapX(16).GapY(8).Wrap().AlignItems(ui.Center).Shrink(0).Children(func() {
		if ly.Synced {
			mode := 0
			if s.set.LyricsPlain {
				mode = 1
			}
			if s.segmented(c, "lyrics-mode-"+where, locales.LyricsShowAs(), &mode, locales.LyricsSynced(), locales.LyricsPlainText()) {
				s.set.LyricsPlain = mode == 1
				s.save()
			}
		}
		if canTranslate {
			on := x.translate
			if s.lyricsSwitch(c, "lyrics-translate-"+where, locales.LyricsTranslate(), locales.LyricsTranslateLyrics(), locales.LyricsTranslateTip(), &on) {
				s.setTranslate(x, lang, on)
			}
		}
		defer share()
		if !canRomanize {
			return
		}
		// The switch and its ways go on one line.
		ui.Row(c).Gap(16).AlignItems(ui.Center).Shrink(0).Children(func() {
			as, tip := romanizeAs(x, lang)
			names := romanizeNames(as == "ja")
			on := x.romanize != ""
			if s.lyricsSwitch(c, "lyrics-romanize-"+where, names[0], locales.LyricsShowInLatin(), tip, &on) {
				s.romanizeLyrics(x, as, on, settings.RomanizeUnder)
			}
			if x.romanize == "" {
				return
			}
			mode := 0
			if x.romanize == settings.RomanizeOnly {
				mode = 1
			}
			if s.segmented(c, "lyrics-romanize-mode-"+where, locales.LyricsShowInLatin(), &mode, names[1], names[2]) {
				how := settings.RomanizeUnder
				if mode == 1 {
					how = settings.RomanizeOnly
				}
				s.romanizeLyrics(x, as, true, how)
			}
		})
	})
}

// lyricsRailWidth is the width of the now playing page's column of lyrics
// choices, which the lyrics keep clear of.
const lyricsRailWidth = 40

// lyricsRail is lyricsSwitches on the now playing page: a column of icon
// buttons at the right of the lyrics of t, lit while on, which leaves the
// lyrics the room above them. Showing in Latin letters under each line or
// in its place is its button's menu.
func (s *shell) lyricsRail(c *ui.Context, t media.Track, ly lyrics.Lyrics, lang string, canTranslate, canRomanize bool, x *lyricsExtra) {
	ui.Column(c.Key("lyrics-rail")).Absolute().Top(0).Bottom(0).Right(0).Width(lyricsRailWidth).Gap(8).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		if ly.Synced {
			plain := s.set.LyricsPlain
			shown := locales.LyricsSynced()
			if plain {
				shown = locales.LyricsPlainText()
			}
			if s.untippedIconButton(c, "lyrics-mode-page", "clock", locales.LyricsShowAs(), !plain, 18).Tooltip(shown).Clicked() {
				s.set.LyricsPlain = !plain
				s.save()
			}
		}
		if canTranslate {
			if s.untippedIconButton(c, "lyrics-translate-page", "translate", locales.LyricsTranslateLyrics(), x.translate, 18).Tooltip(locales.LyricsTranslateTip()).Clicked() {
				s.setTranslate(x, lang, !x.translate)
			}
		}
		if canRomanize {
			as, tip := romanizeAs(x, lang)
			names := romanizeNames(as == "ja")
			b := s.untippedIconButton(c, "lyrics-romanize-page", "romanize", names[0], x.romanize != "", 18).Tooltip(tip)
			b.ContextMenu(func(m *ui.Menu) {
				if m.Item(names[1]).Checked(x.romanize == settings.RomanizeUnder).Chosen() {
					s.romanizeLyrics(x, as, true, settings.RomanizeUnder)
				}
				if m.Item(names[2]).Checked(x.romanize == settings.RomanizeOnly).Chosen() {
					s.romanizeLyrics(x, as, true, settings.RomanizeOnly)
				}
			})
			if b.Clicked() {
				s.romanizeLyrics(x, as, x.romanize == "", settings.RomanizeUnder)
			}
		}
		picking := s.lyricPick.picking(t.ID)
		if s.iconButton(c, "lyrics-share-start-page", "share", locales.LyricsShareLines(), picking, 18).Clicked() {
			s.lyricPick = lyricsPick{}
			if !picking {
				s.lyricPick = lyricsPick{on: true, track: t.ID}
			}
		}
	})
}

// setTranslate translates the lyrics in the language lang, or stops, as
// on says, from now on.
func (s *shell) setTranslate(x *lyricsExtra, lang string, on bool) {
	x.translate = on
	s.set.LyricsTranslate = slices.DeleteFunc(slices.Clone(s.set.LyricsTranslate), func(l string) bool { return l == lang })
	if on {
		s.set.LyricsTranslate = append(s.set.LyricsTranslate, lang)
	}
	s.save()
}

// romanizeAs returns the language whose lyrics x shows in Latin letters,
// of lyrics MixMusic tells are in lang, and the tip of its switch.
// Japanese is romaji, kept as "ja" though MixMusic couldn't tell the
// lyrics' language.
func romanizeAs(x *lyricsExtra, lang string) (string, string) {
	switch {
	case x.ja:
		return "ja", locales.LyricsRomajiTip()
	case unisonFirst[x.local]:
		return x.local, locales.LyricsUnisonFirstTip()
	case x.local != "":
		return x.local, locales.LyricsLocalTip()
	}
	return lang, locales.LyricsRomanizeTip()
}

// romanizeLyrics shows the lyrics of the language as in Latin letters as how
// says, while on, or as they are written, from now on.
func (s *shell) romanizeLyrics(x *lyricsExtra, as string, on bool, how string) {
	x.romanize = ""
	if on {
		x.romanize = how
	}
	s.setRomanize(as, x.romanize)
}

// lyricsSwitch is a switch above the lyrics with its label, and tip,
// which tells what it does: it tells whether the user flipped it.
func (s *shell) lyricsSwitch(c *ui.Context, key, label, name, tip string, on *bool) bool {
	changed := false
	ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
		changed = s.switchOf(c.Key(key), on).Label(name).Tooltip(tip).Changed()
		ui.Text(c, s.lk.sentence(label)).FontSize(s.lk.smallSize())
	})
	return changed
}

// setRomanize shows the lyrics of the language lang in Latin letters as
// how says, or as they are written when how is "".
func (s *shell) setRomanize(lang, how string) {
	m := maps.Clone(s.set.LyricsRomanize)
	if m == nil {
		m = map[string]string{}
	}
	if how == "" {
		delete(m, lang)
	} else {
		m[lang] = how
	}
	s.set.LyricsRomanize = m
	s.save()
}

func (s *shell) lyricsNote(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(s.lk.smallSize()).TextColor(s.lk.muted).Padding(0, 24, 8).Shrink(0)
}

// translateOptions are the choices of "Translate lyrics to", the system's
// language first. They are built on each frame: the language may change.
func (s *shell) translateOptions() []string {
	system := s.systemLanguage()
	for _, l := range translateLanguages {
		if l.code == system {
			system = l.name()
		}
	}
	options := []string{locales.LyricsSettingsSystemLanguage(system)}
	for _, l := range translateLanguages {
		options = append(options, l.name())
	}
	return options
}

// lyricsRows are the rows of Settings › Playback › Lyrics after
// Translate lyrics to, which register from their features' files.
const lyricsRows = "playback › Lyrics"

func init() { addSettings("playback", 60, (*shell).lyricsSettings) }

// lyricsSettings chooses the language lyrics translate to, and shows the
// other rows of the lyrics.
func (s *shell) lyricsSettings(c *ui.Context) {
	s.group(c, locales.LyricsSettingsGroup(), func() {
		s.row(c, locales.LyricsSettingsTranslate(), locales.LyricsSettingsDesc(), func() {
			options := s.translateOptions()
			chosen := options[0]
			for i, l := range translateLanguages {
				if l.code == s.set.LyricsTranslateTo {
					chosen = options[i+1]
				}
			}
			if s.selectOf(c.Key("translate-to"), &chosen, options).Width(220).Label(locales.LyricsSettingsTranslate()).Changed() {
				s.set.LyricsTranslateTo = ""
				if i := slices.Index(options, chosen); i > 0 {
					s.set.LyricsTranslateTo = translateLanguages[i-1].code
				}
				s.save()
			}
		})
		s.row(c, locales.LyricsSettingsLLMURL(), locales.LyricsSettingsLLMDesc(), func() {
			in, _ := s.settingInput(c, "lyrics-llm-url", s.set.LyricsLLMURL, func(v string) {
				if v != "" && !settings.LLMURL(v) {
					// Kept, it would be dropped unseen, and Unison would translate.
					s.notes = append(s.notes, noticeOf(errors.New(locales.LyricsSettingsLLMURLInvalid())))
					return
				}
				s.set.LyricsLLMURL = v
				s.save()
			})
			in.Placeholder("https://api.openai.com/v1").Width(260).Label(locales.LyricsSettingsLLMURL())
		})
		s.row(c, locales.LyricsSettingsLLMModel(), "", func() {
			in, _ := s.settingInput(c, "lyrics-llm-model", s.set.LyricsLLMModel, func(v string) {
				s.set.LyricsLLMModel = v
				s.save()
			})
			in.Placeholder("gpt-4o-mini").Width(260).Label(locales.LyricsSettingsLLMModel())
		})
		s.row(c, locales.AccountSettingsAPIKey(), "", func() {
			// The key never shows: the input stays empty, and says whether one is kept.
			placeholder := locales.LyricsSettingsLLMKeyNone()
			if s.svc.System.HasLyricsLLMKey() {
				placeholder = locales.LyricsSettingsLLMKeyKept()
			}
			in, d := s.settingInput(c, "lyrics-llm-key", "", func(v string) {
				if err := s.svc.System.SetLyricsLLMKey(v); err != nil {
					s.notes = append(s.notes, noticeOf(err))
				}
			})
			in.Password().Placeholder(placeholder).Width(260).Label(locales.AccountSettingsAPIKey())
			if s.svc.System.HasLyricsLLMKey() && s.link(c, "lyrics-llm-key-forget", locales.LyricsSettingsLLMKeyForget()) {
				d.text = ""
				if err := s.svc.System.SetLyricsLLMKey(""); err != nil {
					s.notes = append(s.notes, noticeOf(err))
				}
			}
		})
		s.settingsOf(c, lyricsRows)
	})
}
