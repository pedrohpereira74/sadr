package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	return NewStorage(t.TempDir())
}

func TestUpdateRecordOverwritesInPlace(t *testing.T) {
	s := newTestStorage(t)
	r, _ := model.NewRecordWithOptions("Update me", "full")
	r.Status = "active"
	path, err := s.SaveRecord(r)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	r.Status = "deprecated"
	if err := s.UpdateRecord(path, r); err != nil {
		t.Fatalf("update: %v", err)
	}

	loaded, err := s.LoadRecord(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Status != "deprecated" {
		t.Errorf("expected status deprecated, got %q", loaded.Status)
	}

	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestSaveRecordCreatesFile(t *testing.T) {
	s := newTestStorage(t)
	r, _ := model.NewRecordWithOptions("Use retry", "full")

	path, err := s.SaveRecord(r)
	if err != nil {
		t.Fatalf("Unexpected error = %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist at '%s'", path)
	}
}

func TestSaveRecordCorrectFilename(t *testing.T) {
	s := newTestStorage(t)
	r, _ := model.NewRecordWithOptions("Use retry", "full")

	path, err := s.SaveRecord(r)
	if err != nil {
		t.Fatalf("Unexpected error = %v", err)
	}

	expected := "sadr-record-0001-use-retry.md"
	if filepath.Base(path) != expected {
		t.Errorf("expected filename '%s', got '%s'", expected, filepath.Base(path))
	}
}

func TestSaveRecordVerifySequentialID(t *testing.T) {
	s := newTestStorage(t)

	r1, _ := model.NewRecordWithOptions("first record", "full")
	r2, _ := model.NewRecordWithOptions("second record", "full")

	_, _ = s.SaveRecord(r1)
	path2, err := s.SaveRecord(r2)
	if err != nil {
		t.Fatalf("Unexpected error = %v", err)
	}

	expected := "sadr-record-0002-second-record.md"
	if filepath.Base(path2) != expected {
		t.Errorf("expected filename '%s', got '%s'", expected, filepath.Base(path2))
	}
}

func TestLoadRecordReadsFile(t *testing.T) {
	s := newTestStorage(t)

	r, _ := model.NewRecordWithOptions("Use retry with backoff", "full")
	r.Snippet = "client := retryablehttp.NewClient()"
	r.FileRef = "internal/http/client.go"
	r.Status = "accepted"
	r.Fields["context"] = "The payment service was unreliable"

	path, _ := s.SaveRecord(r)

	loaded, err := s.LoadRecord(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Title != "Use retry with backoff" {
		t.Errorf("expected title 'Use retry with backoff', got '%s'", loaded.Title)
	}
	if loaded.FileRef != "internal/http/client.go" {
		t.Errorf("expected file_ref 'internal/http/client.go', got '%s'", loaded.FileRef)
	}
	if loaded.Snippet != "client := retryablehttp.NewClient()" {
		t.Errorf("expected snippet content, got '%s'", loaded.Snippet)
	}
	if loaded.Status != "accepted" {
		t.Errorf("expected status 'accepted', got '%s'", loaded.Status)
	}
}

func TestListRecords(t *testing.T) {
	s := newTestStorage(t)

	r1, _ := model.NewRecordWithOptions("First record", "full")
	r2, _ := model.NewRecordWithOptions("Second record", "full")

	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	records, err := s.ListRecords()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestDeleteRecord(t *testing.T) {
	s := newTestStorage(t)

	r, _ := model.NewRecordWithOptions("To be deleted", "full")
	path, _ := s.SaveRecord(r)

	err := s.DeleteRecord(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted at %s", path)
	}
}

func TestSlugifyRemovesSpecialChars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Use Redis/Memcached", "use-redismemcached"},
		{"What's next?", "whats-next"},
		{"Hello World", "hello-world"},
		{"API keys & secrets!", "api-keys-secrets"},
		{"  spaces  ", "spaces"},
		{"foo--bar", "foo-bar"},
		{strings.Repeat("a", 200), strings.Repeat("a", 80)},
		{strings.Repeat("word ", 30), "word-word-word-word-word-word-word-word-word-word-word-word-word-word-word-word"},
	}
	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestListRecordsSkipsInvalidFile(t *testing.T) {
	s := newTestStorage(t)

	r, _ := model.NewRecordWithOptions("Valid record", "full")
	_, _ = s.SaveRecord(r)

	invalidPath := filepath.Join(s.Dir, "sadr-9999-bad.md")
	_ = os.WriteFile(invalidPath, []byte("not valid frontmatter at all"), 0644)

	records, err := s.ListRecords()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 valid record, got %d", len(records))
	}
}

func TestParseFileID(t *testing.T) {
	tests := []struct {
		filename string
		want     int
	}{
		{"sadr-record-0001-use-retry.md", 1},
		{"sadr-record-0042-redis-cache.md", 42},
		{"sadr-record-0100-big.md", 100},
		{"sadr-answer-0003-tech-lead.md", 3},
		{"sadr-report-0007-dba.md", 7},
		{"sadr-docs-0002-auth.md", 2},
		{"sadr-0001-use-retry.md", 1},
		{"sadr-0042-redis-cache.md", 42},
		{"not-a-record.md", 0},
		{"sadr-bad-name.md", 0},
	}
	for _, tt := range tests {
		got := ParseFileID(tt.filename)
		if got != tt.want {
			t.Errorf("ParseFileID(%q) = %d, want %d", tt.filename, got, tt.want)
		}
	}
}

func TestGetRecordByFileID(t *testing.T) {
	s := newTestStorage(t)

	r1, _ := model.NewRecordWithOptions("First record", "full")
	r2, _ := model.NewRecordWithOptions("Second record", "full")

	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	r, path, err := s.GetRecordByFileID(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Title != "Second record" {
		t.Errorf("expected title 'Second record', got '%s'", r.Title)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestGetRecordByFileIDNotFound(t *testing.T) {
	s := newTestStorage(t)

	r, _ := model.NewRecordWithOptions("Only record", "full")
	_, _ = s.SaveRecord(r)

	_, _, err := s.GetRecordByFileID(99)
	if err == nil {
		t.Error("expected error for non-existent ID, got nil")
	}
}

func TestListRecordEntries(t *testing.T) {
	s := newTestStorage(t)

	r1, _ := model.NewRecordWithOptions("First", "full")
	r2, _ := model.NewRecordWithOptions("Second", "full")

	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	entries, err := s.ListRecordEntries()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].FileID != 1 {
		t.Errorf("expected first entry FileID 1, got %d", entries[0].FileID)
	}
	if entries[1].FileID != 2 {
		t.Errorf("expected second entry FileID 2, got %d", entries[1].FileID)
	}
}

func TestSaveRecordConcurrent(t *testing.T) {
	s := newTestStorage(t)
	const n = 20

	errs := make(chan error, n)
	for i := range n {
		go func(i int) {
			r, _ := model.NewRecordWithOptions(fmt.Sprintf("concurrent record %d", i), "full")
			_, err := s.SaveRecord(r)
			errs <- err
		}(i)
	}

	for range n {
		if err := <-errs; err != nil {
			t.Errorf("concurrent SaveRecord failed: %v", err)
		}
	}

	records, err := s.ListRecords()
	if err != nil {
		t.Fatalf("unexpected error listing records: %v", err)
	}
	if len(records) != n {
		t.Errorf("expected %d records after concurrent writes, got %d", n, len(records))
	}

	seen := map[int]bool{}
	entries, _ := s.ListRecordEntries()
	for _, e := range entries {
		if seen[e.FileID] {
			t.Errorf("duplicate file ID %d found", e.FileID)
		}
		seen[e.FileID] = true
	}
}

func TestFormatBodyStatusRendered(t *testing.T) {
	s := newTestStorage(t)
	r, _ := model.NewRecordWithOptions("Status test", "full")
	r.Status = "accepted"

	path, err := s.SaveRecord(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error reading file: %v", err)
	}

	if !strings.Contains(string(data), "**Status:** `#accepted`") {
		t.Errorf("expected status formatted as `#accepted`, got:\n%s", string(data))
	}
}

func TestFormatBodySnippetRenderedAfterFields(t *testing.T) {
	s := newTestStorage(t)
	r, _ := model.NewRecordWithOptions("Ordering test", "full")
	r.FieldOrder = []string{"context", "decision"}
	r.Fields = map[string]string{
		"context":  "Why we did it.",
		"decision": "What we chose.",
	}
	r.FineTuningHint = "focus on rollback"
	r.Snippet = "func main() {}"

	path, err := s.SaveRecord(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error reading file: %v", err)
	}
	out := string(data)

	snippetIdx := strings.Index(out, "## Snippet")
	contextIdx := strings.Index(out, "## Context")
	decisionIdx := strings.Index(out, "## Decision")
	questionIdx := strings.Index(out, "**Question:**")

	if snippetIdx == -1 || contextIdx == -1 || decisionIdx == -1 || questionIdx == -1 {
		t.Fatalf("expected all sections present, got:\n%s", out)
	}
	if !(contextIdx < decisionIdx && decisionIdx < questionIdx && questionIdx < snippetIdx) {
		t.Errorf("expected order context < decision < question < snippet, got:\n%s", out)
	}
}

func TestLoadRecordPreservesStatus(t *testing.T) {
	s := newTestStorage(t)
	r, _ := model.NewRecordWithOptions("Status roundtrip", "full")
	r.Status = "accepted"

	path, _ := s.SaveRecord(r)
	loaded, err := s.LoadRecord(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Status != "accepted" {
		t.Errorf("expected status 'accepted', got '%s'", loaded.Status)
	}
}

func TestCapitalizeKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"context", "Context"},
		{"file_ref", "File Ref"},
		{"api_key", "Api Key"},
		{"", ""},
	}
	for _, tt := range tests {
		got := capitalizeKey(tt.input)
		if got != tt.want {
			t.Errorf("capitalizeKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
