package search

import (
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func TestSearchByTitle(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r1, _ := model.NewRecord("Use retry with backoff")
	r2, _ := model.NewRecord("Redis cache strategy")

	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	results, err := Search(dir, "retry", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearchDeepFindsInSnippet(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r, _ := model.NewRecord("HTTP client setup")
	r.Snippet = "retryablehttp.NewClient()"

	_, _ = s.SaveRecord(r)

	shallow, _ := Search(dir, "retryablehttp", false)
	if len(shallow) != 0 {
		t.Errorf("expected 0 results without deep, got %d", len(shallow))
	}

	deep, err := Search(dir, "retryablehttp", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deep) != 1 {
		t.Errorf("expected 1 result with deep, got %d", len(deep))
	}
}

func TestSearchByTags(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r1, _ := model.NewRecord("Cache strategy")
	r1.Fields["tags"] = "database,performance"

	r2, _ := model.NewRecord("Auth flow")
	r2.Fields["tags"] = "security,api"

	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	results, err := Search(dir, "security", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage(dir)

	r, _ := model.NewRecord("Use Retry With Backoff")
	_, _ = s.SaveRecord(r)

	results, err := Search(dir, "retry", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}
