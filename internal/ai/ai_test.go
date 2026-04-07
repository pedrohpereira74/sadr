package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	snippet := "client := retryablehttp.NewClient()\nclient.RetryMax = 3"
	fields := []string{"title", "tags", "context", "decision"}
	language := "english"

	prompt := BuildPrompt(snippet, fields, language, false)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}

func TestParseResponseWithArrays(t *testing.T) {
	response := `{
		"title": "Use retry with backoff",
		"tags": ["api", "performance"]
	}`

	result, err := ParseResponse(response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["tags"] != "api, performance" {
		t.Errorf("expected 'api, performance', got '%s'", result["tags"])
	}
}

type testRoundTripper struct {
	handler http.HandlerFunc
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.handler(rec, req)
	return rec.Result(), nil
}

func mockClient(handler http.HandlerFunc) *http.Client {
	return &http.Client{Transport: &testRoundTripper{handler: handler}}
}

func TestSuggestValidResponse(t *testing.T) {
	old := httpClient
	httpClient = mockClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"title\":\"Use retry\"}"}]}}]}`))
	})
	defer func() { httpClient = old }()

	result, err := Suggest(context.Background(), "some code", []string{"title"}, "english", "fake-key", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["title"] != "Use retry" {
		t.Errorf("expected 'Use retry', got '%s'", result["title"])
	}
}

func TestSuggestAPIError(t *testing.T) {
	old := httpClient
	httpClient = mockClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	})
	defer func() { httpClient = old }()

	_, err := Suggest(context.Background(), "code", []string{"title"}, "english", "bad-key", "", false)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestSuggestEmptyResponse(t *testing.T) {
	old := httpClient
	httpClient = mockClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"candidates":[]}`))
	})
	defer func() { httpClient = old }()

	_, err := Suggest(context.Background(), "code", []string{"title"}, "english", "fake-key", "", false)
	if err == nil {
		t.Fatal("expected error for empty candidates")
	}
}

func TestSuggestInvalidJSON(t *testing.T) {
	old := httpClient
	httpClient = mockClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`not json at all`))
	})
	defer func() { httpClient = old }()

	_, err := Suggest(context.Background(), "code", []string{"title"}, "english", "fake-key", "", false)
	if err == nil {
		t.Fatal("expected error for invalid JSON response body")
	}
}

func TestSuggestMissingAPIKey(t *testing.T) {
	t.Setenv("AI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	_, err := Suggest(context.Background(), "code", []string{"title"}, "english", "", "", false)
	if err == nil {
		t.Fatal("expected error when no API key is provided")
	}
}

func TestGenerateText(t *testing.T) {
	old := httpClient
	httpClient = mockClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"# Report\n\nThis is a free-form report."}]}}]}`))
	})
	defer func() { httpClient = old }()

	text, err := GenerateText(context.Background(), "analyze this", "fake-key", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "# Report\n\nThis is a free-form report." {
		t.Errorf("unexpected text: %s", text)
	}
}

func TestGenerateTextAPIError(t *testing.T) {
	old := httpClient
	httpClient = mockClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	})
	defer func() { httpClient = old }()

	_, err := GenerateText(context.Background(), "prompt", "fake-key", "", 0)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestGenerateTextMissingKey(t *testing.T) {
	t.Setenv("AI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	_, err := GenerateText(context.Background(), "prompt", "", "", 0)
	if err == nil {
		t.Fatal("expected error when no API key is provided")
	}
}

func TestSuggestSetsAuthHeader(t *testing.T) {
	var gotKey string
	old := httpClient
	httpClient = mockClient(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-goog-api-key")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"title\":\"x\"}"}]}}]}`))
	})
	defer func() { httpClient = old }()

	_, _ = Suggest(context.Background(), "code", []string{"title"}, "english", "my-secret-key", "", false)
	if gotKey != "my-secret-key" {
		t.Errorf("expected API key header 'my-secret-key', got '%s'", gotKey)
	}
}
