package search

import (
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func loadRecords(t *testing.T, dir string) []model.Record {
	t.Helper()
	s := storage.NewStorage(dir)
	records, err := s.ListRecords()
	if err != nil {
		t.Fatalf("failed to load records: %v", err)
	}
	return records
}

func TestSearchByTitle(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r1, _ := model.NewRecordWithOptions("Use retry with backoff", "full")
	r2, _ := model.NewRecordWithOptions("Redis cache strategy", "full")

	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	records := loadRecords(t, dir)
	results := Search(records, "retry", false)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearchDeepFindsInSnippet(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r, _ := model.NewRecordWithOptions("HTTP client setup", "full")
	r.Snippet = "retryablehttp.NewClient()"

	_, _ = s.SaveRecord(r)

	records := loadRecords(t, dir)
	shallow := Search(records, "retryablehttp", false)
	if len(shallow) != 0 {
		t.Errorf("expected 0 results without deep, got %d", len(shallow))
	}

	deep := Search(records, "retryablehttp", true)
	if len(deep) != 1 {
		t.Errorf("expected 1 result with deep, got %d", len(deep))
	}
}

func TestSearchByTags(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r1, _ := model.NewRecordWithOptions("Cache strategy", "full")
	r1.Fields["tags"] = "database,performance"

	r2, _ := model.NewRecordWithOptions("Auth flow", "full")
	r2.Fields["tags"] = "security,api"

	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	records := loadRecords(t, dir)
	results := Search(records, "security", false)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r, _ := model.NewRecordWithOptions("Use Retry With Backoff", "full")
	_, _ = s.SaveRecord(r)

	records := loadRecords(t, dir)
	results := Search(records, "retry", false)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}
