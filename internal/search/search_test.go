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

func filterRecords(records []model.Record, query string, deep bool) []model.Record {
	var results []model.Record
	for _, r := range records {
		if Matches(r, query, deep) {
			results = append(results, r)
		}
	}
	if results == nil {
		return []model.Record{}
	}
	return results
}

func TestSearchByTitle(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r1, _ := model.NewRecordWithOptions("Use retry with backoff", "full")
	r2, _ := model.NewRecordWithOptions("Redis cache strategy", "full")

	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	records := loadRecords(t, dir)
	results := filterRecords(records, "retry", false)
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
	shallow := filterRecords(records, "retryablehttp", false)
	if len(shallow) != 0 {
		t.Errorf("expected 0 results without deep, got %d", len(shallow))
	}

	deep := filterRecords(records, "retryablehttp", true)
	if len(deep) != 1 {
		t.Errorf("expected 1 result with deep, got %d", len(deep))
	}
}

func TestSearchByTags(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r1, _ := model.NewRecordWithOptions("Cache strategy", "full")
	r1.Tags = []string{"database", "performance"}

	r2, _ := model.NewRecordWithOptions("Auth flow", "full")
	r2.Tags = []string{"security", "api"}

	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	records := loadRecords(t, dir)
	results := filterRecords(records, "security", false)
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
	results := filterRecords(records, "retry", false)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearchNoResultsReturnsEmptySlice(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r, _ := model.NewRecordWithOptions("Use retry", "full")
	_, _ = s.SaveRecord(r)

	records := loadRecords(t, dir)
	results := filterRecords(records, "banana", false)
	if results == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchDeepFindsInCustomFields(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r, _ := model.NewRecordWithOptions("Payment service", "full")
	r.Fields["context"] = "The unreliable external vendor caused repeated timeouts"

	_, _ = s.SaveRecord(r)

	records := loadRecords(t, dir)
	shallow := filterRecords(records, "vendor", false)
	if len(shallow) != 0 {
		t.Errorf("expected 0 results without deep, got %d", len(shallow))
	}

	deep := filterRecords(records, "vendor", true)
	if len(deep) != 1 {
		t.Errorf("expected 1 result with deep, got %d", len(deep))
	}
}

func TestMatchesReturnsFalseOnNoMatch(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Use retry", "full")
	if Matches(r, "banana", true) {
		t.Error("expected Matches to return false for non-matching query")
	}
}

func TestMatchesDeepFindsInSnippet(t *testing.T) {
	r, _ := model.NewRecordWithOptions("HTTP client", "full")
	r.Snippet = "retryablehttp.NewClient()"
	if !Matches(r, "retryablehttp", true) {
		t.Error("expected Matches to find content in snippet with deep=true")
	}
	if Matches(r, "retryablehttp", false) {
		t.Error("expected Matches to miss snippet content with deep=false")
	}
}
