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

func TestRenderRecordSnippetAfterFields(t *testing.T) {
	data := ExportData{
		Title:    "Ordering",
		Type:     "full",
		Snippet:  "func main() {}",
		Question: "focus on rollback",
		Fields: []ExportField{
			{Key: "Context", Value: "Why we did it"},
			{Key: "Decision", Value: "What we chose"},
		},
	}

	html, err := RenderRecord(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	contextIdx := strings.Index(html, "Why we did it")
	decisionIdx := strings.Index(html, "What we chose")
	questionIdx := strings.Index(html, "focus on rollback")
	snippetIdx := strings.Index(html, "<h2>Snippet</h2>")

	if contextIdx == -1 || decisionIdx == -1 || questionIdx == -1 || snippetIdx == -1 {
		t.Fatalf("expected all sections present, got:\n%s", html)
	}
	if !(contextIdx < decisionIdx && decisionIdx < questionIdx && questionIdx < snippetIdx) {
		t.Errorf("expected fields before question before snippet, got:\n%s", html)
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

func TestRenderRecordWithStatusAndQuestion(t *testing.T) {
	data := ExportData{
		Title:    "Auth refactor",
		Type:     "full",
		Status:   "proposed",
		Question: "focus on security implications",
	}

	html, err := RenderRecord(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(html, "<strong>Status:</strong> proposed") {
		t.Error("expected status block in HTML")
	}
	if !strings.Contains(html, "<strong>Question:</strong> focus on security implications") {
		t.Error("expected question block in HTML")
	}
}

func TestRenderRecordWithoutStatusAndQuestion(t *testing.T) {
	data := ExportData{
		Title: "No extras",
		Type:  "adr",
	}

	html, err := RenderRecord(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(html, "Status:") {
		t.Error("expected no status block when status is empty")
	}
	if strings.Contains(html, "Question:") {
		t.Error("expected no question block when question is empty")
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
