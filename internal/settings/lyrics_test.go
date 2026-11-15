package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestLyricsTranslation keeps the languages lyrics translate and romanize
// in, and leaves out what isn't one.
func TestLyricsTranslation(t *testing.T) {
	s := Settings{
		Lyrics: Lyrics{
			LyricsTranslate: []string{"ja", "", "ja", "zh-Hant", "Japanese", "x"},
			LyricsRomanize:  map[string]string{"ja": RomanizeOnly, "ko": RomanizeUnder, "ru": "sideways", "<b>": RomanizeOnly},
		},
		Translation: Translation{LyricsTranslateTo: "zh-Hant"},
	}.Clean()
	if strings.Join(s.LyricsTranslate, ",") != "ja,,zh-Hant" {
		t.Errorf("translate = %q", s.LyricsTranslate)
	}
	if len(s.LyricsRomanize) != 2 || s.LyricsRomanize["ja"] != RomanizeOnly || s.LyricsRomanize["ko"] != RomanizeUnder {
		t.Errorf("romanize = %v", s.LyricsRomanize)
	}
	if s.LyricsTranslateTo != "zh-Hant" {
		t.Errorf("translate to %q", s.LyricsTranslateTo)
	}
	for _, to := range []string{"english", "EN", "en-us", "../x"} {
		if got := (Settings{Translation: Translation{LyricsTranslateTo: to}}).Clean().LyricsTranslateTo; got != "" {
			t.Errorf("translate to %q kept as %q", to, got)
		}
	}
	// Settings from before translations translate nothing, to the
	// system's language.
	var before Settings
	if err := json.Unmarshal([]byte(`{"theme":"mix"}`), &before); err != nil {
		t.Fatal(err)
	}
	if c := before.Clean(); len(c.LyricsTranslate) != 0 || c.LyricsTranslate == nil || c.LyricsRomanize == nil || c.LyricsTranslateTo != "" {
		t.Errorf("settings from before: %+v", c)
	}
}

// TestLyricsLLM keeps the server of a language model only when it is a
// URL of the web, and translates with it only with a model too.
func TestLyricsLLM(t *testing.T) {
	for url, want := range map[string]string{
		" https://api.openai.com/v1 ": "https://api.openai.com/v1",
		"http://localhost:11434/v1":   "http://localhost:11434/v1",
		"http://127.0.0.1:1234/v1":    "http://127.0.0.1:1234/v1",
		"http://[::1]:1234/v1":        "http://[::1]:1234/v1",
		"http://api.example.com/v1":   "",
		"api.openai.com/v1":           "",
		"file:///etc/passwd":          "",
		"https://":                    "",
	} {
		if got := (Settings{Translation: Translation{LyricsLLMURL: url}}).Clean().LyricsLLMURL; got != want {
			t.Errorf("%q kept as %q", url, got)
		}
	}
	if (Translation{LyricsLLMURL: "https://x"}).LLM() || !(Translation{LyricsLLMURL: "https://x", LyricsLLMModel: "m"}).LLM() {
		t.Error("LLM wrong")
	}
}
