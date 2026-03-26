package ai

import (
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
