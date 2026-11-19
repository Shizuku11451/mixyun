package lyrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// llmTimeout bounds a translation by a language model, which writes the
// whole song before it answers.
const llmTimeout = time.Minute

// LLM translates lyrics with a language model the user chose, on a server
// that speaks OpenAI's chat completions, as OpenAI, DeepSeek, OpenRouter,
// Ollama and LM Studio do.
type LLM struct {
	HTTP *http.Client
	// URL is the server's base URL, as "https://api.openai.com/v1", or
	// its whole endpoint, ending in /chat/completions.
	URL, Key, Model string
}

// llmPrompt tells the model what to write.
const llmPrompt = `You translate song lyrics. The user sends a JSON array of lines. ` +
	`Answer with JSON only, {"lines":[{"t":"…","r":"…"}]}, one object per line, in order, as many as were sent. ` +
	`"t" is the line translated into the language %[1]s, faithful to its meaning and its tone, as a singer would say it. ` +
	`A line that mixes languages keeps every word already in %[1]s exactly as written, in its script and its place, ` +
	`and only the rest is translated, in the line's order and with its punctuation. ` +
	`"t" is "" for a blank line or one with nothing to translate. "r" is the line in Latin letters, by the usual romanization ` +
	`of its language (romaji, pinyin, Revised Romanization…), its words in Latin letters as they are; "" for a line in Latin letters already.`

// Translate translates lines of lyrics into the language to, a code as
// "en" or "zh-Hant", and romanizes them. from is their language, "" when
// unknown, which the model tells for itself.
func (l LLM) Translate(ctx context.Context, lines []string, to, from string) (Translation, error) {
	t := Translation{From: from, To: to, Lines: make([]TranslatedLine, len(lines))}
	if strings.TrimSpace(strings.Join(lines, "")) == "" {
		return t, nil
	}
	sent, err := json.Marshal(lines)
	if err != nil {
		return Translation{}, err
	}
	// shortcut: the whole song in one request; lyrics past a few hundred lines may outgrow small models' output, chunk them if users hit it.
	body, err := json.Marshal(map[string]any{
		"model":       l.Model,
		"temperature": 0.3,
		"messages": []map[string]string{
			{"role": "system", "content": fmt.Sprintf(llmPrompt, to)},
			{"role": "user", "content": string(sent)},
		},
	})
	if err != nil {
		return Translation{}, err
	}
	endpoint := strings.TrimRight(l.URL, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Translation{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.Key != "" {
		req.Header.Set("Authorization", "Bearer "+l.Key)
	}
	client := http.Client{Timeout: llmTimeout}
	if l.HTTP != nil {
		client = *l.HTTP
	}
	res, err := client.Do(req)
	if err != nil {
		return Translation{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 300))
		if res.StatusCode == http.StatusTooManyRequests {
			return Translation{}, fmt.Errorf("%w: HTTP 429: %s", ErrRateLimited, msg)
		}
		return Translation{}, fmt.Errorf("HTTP %d: %s", res.StatusCode, msg)
	}
	var ans struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(res.Body).Decode(&ans); err != nil {
		return Translation{}, err
	}
	if len(ans.Choices) == 0 {
		return Translation{}, fmt.Errorf("the model answered nothing")
	}
	content := ans.Choices[0].Message.Content
	// Models wrap JSON in code fences or words: the object is between its braces.
	start, end := strings.Index(content, "{"), strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return Translation{}, fmt.Errorf("the model answered no JSON")
	}
	var got struct {
		Lines []struct {
			T string `json:"t"`
			R string `json:"r"`
		} `json:"lines"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &got); err != nil {
		return Translation{}, fmt.Errorf("the model's JSON: %w", err)
	}
	if len(got.Lines) != len(lines) {
		return Translation{}, fmt.Errorf("the model translated %d lines of %d", len(got.Lines), len(lines))
	}
	for i, g := range got.Lines {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if !same(g.T, lines[i]) {
			t.Lines[i].Text = strings.TrimSpace(g.T)
		}
		if !same(g.R, lines[i]) {
			t.Lines[i].Romanized = strings.TrimSpace(g.R)
		}
	}
	return t, nil
}
