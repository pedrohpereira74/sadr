package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	return NewStorage(t.TempDir())
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

	expected := "sadr-0001-use-retry.md"
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

	expected := "sadr-0002-second-record.md"
	if filepath.Base(path2) != expected {
		t.Errorf("expected filename '%s', got '%s'", expected, filepath.Base(path2))
	}
}

func TestLoadRecordReadsFile(t *testing.T) {
	s := newTestStorage(t)

	r, _ := model.NewRecordWithOptions("Use retry with backoff", "full")
	r.Snippet = "client := retryablehttp.NewClient()"
	r.FileRef = "internal/http/client.go"
	r.Fields["status"] = "accepted"
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
	if loaded.Fields["status"] != "accepted" {
		t.Errorf("expected status 'accepted', got '%s'", loaded.Fields["status"])
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

	// Write an invalid .md file
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
