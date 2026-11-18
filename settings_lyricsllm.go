package main

import (
	"errors"
	"slices"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/settings"
	"github.com/fzn0x/mixmusic/locales"
)

// llmPresets are the servers Translate with offers, each that speaks
// OpenAI's chat completions at its base URL. Ollama and LM Studio run on
// this computer and take no key.
var llmPresets = []struct{ name, url string }{
	{"OpenAI", "https://api.openai.com/v1"},
	{"OpenRouter", "https://openrouter.ai/api/v1"},
	{"Google Gemini", "https://generativelanguage.googleapis.com/v1beta/openai"},
	{"DeepSeek", "https://api.deepseek.com/v1"},
	{"Groq", "https://api.groq.com/openai/v1"},
	{"Mistral", "https://api.mistral.ai/v1"},
	{"xAI", "https://api.x.ai/v1"},
	{"Together AI", "https://api.together.xyz/v1"},
	{"NVIDIA NIM", "https://integrate.api.nvidia.com/v1"},
	{"Ollama", "http://localhost:11434/v1"},
	{"LM Studio", "http://localhost:1234/v1"},
}

func init() { addSettings(lyricsRows, 5, (*shell).lyricsLLMRows) }

// lyricsLLMRows choose what translates lyrics: Unison, or a language model
// on a server of the user's, with its model and API key.
func (s *shell) lyricsLLMRows(c *ui.Context) {
	s.row(c, locales.LyricsSettingsTranslateWith(), locales.LyricsSettingsTranslateWithDesc(), func() {
		options := []string{"Unison"}
		chosen := options[0]
		for _, p := range llmPresets {
			options = append(options, p.name)
			if p.url == s.set.LyricsLLMURL {
				chosen = p.name
			}
		}
		// A server of no preset shows as Custom; it is chosen by typing it.
		if s.set.LyricsLLMURL != "" && chosen == options[0] {
			chosen = locales.LyricsSettingsLLMCustom()
			options = append(options, chosen)
		}
		if s.selectOf(c.Key("lyrics-llm-with"), &chosen, options).Width(220).Label(locales.LyricsSettingsTranslateWith()).Changed() {
			s.set.LyricsLLMURL = ""
			if i := slices.Index(options, chosen); i > 0 && i <= len(llmPresets) {
				s.set.LyricsLLMURL = llmPresets[i-1].url
			}
			delete(s.drafts, "lyrics-llm-url")
			s.save()
		}
	}, "AI", "LLM", "language model", "OpenAI", "Ollama")
	if s.set.LyricsLLMURL == "" {
		return
	}
	s.row(c, locales.LyricsSettingsLLMURL(), locales.LyricsSettingsLLMURLDesc(), func() {
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
	s.row(c, locales.LyricsSettingsLLMModel(), locales.LyricsSettingsLLMModelDesc(), func() {
		in, _ := s.settingInput(c, "lyrics-llm-model", s.set.LyricsLLMModel, func(v string) {
			s.set.LyricsLLMModel = v
			s.save()
		})
		in.Placeholder("gpt-4o-mini").Width(260).Label(locales.LyricsSettingsLLMModel())
	})
	s.row(c, locales.LyricsSettingsLLMKey(), locales.LyricsSettingsLLMKeyDesc(), func() {
		// The key never shows: the input stays empty, and says whether one is kept.
		kept := s.svc.System.HasLyricsLLMKey()
		placeholder := locales.LyricsSettingsLLMKeyNone()
		if kept {
			placeholder = locales.LyricsSettingsLLMKeyKept()
		}
		in, d := s.settingInput(c, "lyrics-llm-key", "", func(v string) {
			if err := s.svc.System.SetLyricsLLMKey(v); err != nil {
				s.notes = append(s.notes, noticeOf(err))
			}
		})
		in.Password().Placeholder(placeholder).Width(260).Label(locales.LyricsSettingsLLMKey())
		if kept && s.link(c, "lyrics-llm-key-forget", locales.LyricsSettingsLLMKeyForget()) {
			d.text = ""
			if err := s.svc.System.SetLyricsLLMKey(""); err != nil {
				s.notes = append(s.notes, noticeOf(err))
			}
		}
	})
}
