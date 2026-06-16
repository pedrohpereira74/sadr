package ai

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestProviderByName(t *testing.T) {
	ok := []string{"", "gemini", "google", "claude", "anthropic", "openai", "gpt", "deepseek", "DeepSeek"}
	for _, name := range ok {
		if _, err := providerByName(name); err != nil {
			t.Errorf("provider %q should be supported: %v", name, err)
		}
	}
	if _, err := providerByName("copilot"); err == nil {
		t.Error("expected unknown provider 'copilot' to error")
	}
}

func TestProviderEndpointsAndAuth(t *testing.T) {
	gem, _ := providerByName("gemini")
	if !strings.Contains(gem.url("m"), "generativelanguage.googleapis.com") {
		t.Error("gemini url wrong")
	}
	if gem.headers("k")["x-goog-api-key"] != "k" {
		t.Error("gemini should auth via x-goog-api-key")
	}

	cl, _ := providerByName("claude")
	if cl.url("m") != "https://api.anthropic.com/v1/messages" {
		t.Errorf("claude url wrong: %s", cl.url("m"))
	}
	h := cl.headers("k")
	if h["x-api-key"] != "k" || h["anthropic-version"] == "" {
		t.Errorf("claude headers wrong: %v", h)
	}

	oa, _ := providerByName("openai")
	if !strings.HasSuffix(oa.url("m"), "/chat/completions") {
		t.Error("openai url should be chat/completions")
	}
	if oa.headers("k")["Authorization"] != "Bearer k" {
		t.Error("openai should use bearer auth")
	}

	ds, _ := providerByName("deepseek")
	if !strings.Contains(ds.url("m"), "deepseek.com") {
		t.Error("deepseek base url wrong")
	}
}

func TestProviderParse(t *testing.T) {
	cl, _ := providerByName("claude")
	got, err := cl.parse([]byte(`{"content":[{"type":"text","text":"hi from claude"}]}`))
	if err != nil || got != "hi from claude" {
		t.Errorf("claude parse: %q, %v", got, err)
	}

	oa, _ := providerByName("openai")
	got, err = oa.parse([]byte(`{"choices":[{"message":{"content":"hi from openai"}}]}`))
	if err != nil || got != "hi from openai" {
		t.Errorf("openai parse: %q, %v", got, err)
	}

	if _, err := oa.parse([]byte(`{"choices":[]}`)); err == nil {
		t.Error("expected error on empty choices")
	}
}

func TestGenerateTextDispatchesToClaude(t *testing.T) {
	old := httpClient
	var gotURL, gotAuthVersion string
	httpClient = mockClient(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		gotAuthVersion = r.Header.Get("anthropic-version")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"claude says hi"}]}`))
	})
	defer func() { httpClient = old }()

	out, err := GenerateText(context.Background(), "claude", "hello", "key", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "claude says hi" {
		t.Errorf("expected claude text, got %q", out)
	}
	if !strings.Contains(gotURL, "anthropic.com") || gotAuthVersion == "" {
		t.Errorf("expected anthropic request, url=%s version=%s", gotURL, gotAuthVersion)
	}
}

func TestGenerateTextUnknownProvider(t *testing.T) {
	if _, err := GenerateText(context.Background(), "nope", "p", "k", "", 0); err == nil {
		t.Error("expected error for unknown provider")
	}
}
