package templates

import (
	"strings"
	"testing"
)

func TestRenderRecordBasic(t *testing.T) {
	data := ExportData{
		Title:      "Use retry with backoff",
		Type:       "full",
		FileRef:    "internal/http/client.go",
		HasFileRef: true,
		Snippet:    "client := retryablehttp.NewClient()",
		Tags:       "api,performance",
		Fields: []ExportField{
			{Key: "context", Value: "HTTP calls were failing silently"},
		},
	}

	html, err := RenderRecord(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{
		"Use retry with backoff",
		"sadr-type",
		"internal/http/client.go",
		"api,performance",
		"retryablehttp.NewClient()",
		"HTTP calls were failing silently",
		"hljs.highlightAll()",
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Errorf("expected HTML to contain %q", want)
		}
	}
}

func TestRenderRecordWithoutOptionalFields(t *testing.T) {
	data := ExportData{
		Title: "Minimal record",
		Type:  "adr",
	}

	html, err := RenderRecord(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(html, "Minimal record") {
		t.Error("expected title in HTML")
	}
	if strings.Contains(html, "<pre><code>") {
		t.Error("expected no snippet block when snippet is empty")
	}
	if strings.Contains(html, "sadr-file-ref") {
		t.Error("expected no file-ref meta when HasFileRef is false")
	}
}

func TestRenderRecordEscapesHTML(t *testing.T) {
	data := ExportData{
		Title:   "<script>alert('xss')</script>",
		Type:    "full",
		Snippet: "x := 1 < 2 && 3 > 1",
		Tags:    "<b>bold</b>",
	}

	html, err := RenderRecord(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(html, "<script>alert") {
		t.Error("expected title to be HTML-escaped")
	}
	if strings.Contains(html, "<b>bold</b>") {
		t.Error("expected tags to be HTML-escaped")
	}
}
