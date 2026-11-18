package main

import "testing"

// TestLLMCustom shows the server's input once Translate with is Custom,
// before a server is typed, and leaves Custom for a preset.
func TestLLMCustom(t *testing.T) {
	s := newTestShell(t)
	tt := testerOf(t, s, 1280, 1800)
	show(s, tt, settingsPath("lyrics", ""))
	if tt.HasText("Server") {
		t.Fatalf("Unison shows the server: %q", tt.Texts())
	}
	click(t, s, tt, "Unison") // the select, showing its choice
	click(t, s, tt, "Custom")
	if !s.llmCustom || s.set.LyricsLLMURL != "" || !tt.HasText("Server") {
		t.Fatalf("Custom: custom %v, URL %q, shows %q", s.llmCustom, s.set.LyricsLLMURL, tt.Texts())
	}
	click(t, s, tt, "Custom")
	click(t, s, tt, "DeepSeek")
	if s.llmCustom || s.set.LyricsLLMURL != "https://api.deepseek.com/v1" {
		t.Errorf("a preset after Custom: custom %v, URL %q", s.llmCustom, s.set.LyricsLLMURL)
	}
}
