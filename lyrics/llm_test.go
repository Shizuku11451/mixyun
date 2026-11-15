package lyrics

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// llmServer answers chat completions with content, and checks the request.
func llmServer(t *testing.T, status int, content string) LLM {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model    string `json:"model"`
			Messages []struct{ Role, Content string }
		}
		json.NewDecoder(r.Body).Decode(&req)
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer k" || req.Model != "m" ||
			len(req.Messages) != 2 || !strings.Contains(req.Messages[0].Content, "zh-Hant") {
			t.Errorf("request %s %v %+v", r.URL.Path, r.Header, req)
		}
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
	}))
	t.Cleanup(srv.Close)
	return LLM{URL: srv.URL + "/v1/", Key: "k", Model: "m"}
}

func TestLLMTranslate(t *testing.T) {
	l := llmServer(t, 200, "```json\n{\"lines\":[{\"t\":\"風箏\",\"r\":\"tako\"},{\"t\":\"\",\"r\":\"\"},{\"t\":\"Hello\",\"r\":\"\"}]}\n```")
	tr, err := l.Translate(context.Background(), []string{"凧", "", "Hello"}, "zh-Hant", "ja")
	if err != nil {
		t.Fatal(err)
	}
	want := []TranslatedLine{{Text: "風箏", Romanized: "tako"}, {}, {}}
	if tr.From != "ja" || tr.To != "zh-Hant" || len(tr.Lines) != 3 || tr.Lines[0] != want[0] || tr.Lines[1] != want[1] || tr.Lines[2] != want[2] {
		t.Errorf("translation = %+v", tr)
	}
	// A model that loses lines is wrong, not half right.
	if _, err := llmServer(t, 200, `{"lines":[{"t":"x"}]}`).Translate(context.Background(), []string{"a", "b"}, "zh-Hant", ""); err == nil {
		t.Error("lines lost, no error")
	}
	if _, err := llmServer(t, 429, "").Translate(context.Background(), []string{"a"}, "zh-Hant", ""); !errors.Is(err, ErrRateLimited) {
		t.Errorf("429: %v", err)
	}
}
