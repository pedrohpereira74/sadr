package hub

import (
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func newModel(entries []storage.RecordEntry) *Model {
	m := New(Options{})
	m.allEntries = entries
	return m
}

func entry(title string, tags []string, snippet string) storage.RecordEntry {
	return storage.RecordEntry{
		Record: model.Record{
			Title:   title,
			Tags:    tags,
			Snippet: snippet,
		},
	}
}

func titlesOf(entries []storage.RecordEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Record.Title
	}
	return out
}

func TestApplyFilterEmptyQuery(t *testing.T) {
	entries := []storage.RecordEntry{
		entry("Alpha", nil, ""),
		entry("Beta", nil, ""),
		entry("Gamma", nil, ""),
	}
	m := newModel(entries)
	m.applyFilter()

	if len(m.filtered) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(m.filtered))
	}
	if m.deepContexts != nil {
		t.Error("deepContexts should be nil for empty query")
	}
}

func TestApplyFilterFuzzyTitle(t *testing.T) {
	entries := []storage.RecordEntry{
		entry("Architecture Decision Record", nil, ""),
		entry("Unrelated document", nil, ""),
	}
	m := newModel(entries)
	m.input.SetValue("arch")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(m.filtered), titlesOf(m.filtered))
	}
	if m.filtered[0].Record.Title != "Architecture Decision Record" {
		t.Errorf("unexpected match: %q", m.filtered[0].Record.Title)
	}
}

func TestApplyFilterTagFallback(t *testing.T) {
	entries := []storage.RecordEntry{
		entry("Unrelated title", []string{"golang", "backend"}, ""),
		entry("Another unrelated", []string{"frontend"}, ""),
	}
	m := newModel(entries)
	m.input.SetValue("golang")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(m.filtered))
	}
	if m.filtered[0].Record.Title != "Unrelated title" {
		t.Errorf("unexpected match: %q", m.filtered[0].Record.Title)
	}
}

func TestApplyFilterNoDuplicateWhenFuzzyAndTagBothMatch(t *testing.T) {
	entries := []storage.RecordEntry{
		entry("golang service", []string{"golang"}, ""),
	}
	m := newModel(entries)
	m.input.SetValue("golang")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 result (no duplicate), got %d", len(m.filtered))
	}
}

func TestApplyFilterDeepMode(t *testing.T) {
	entries := []storage.RecordEntry{
		entry("Some title", nil, "contains the keyword here"),
		entry("Other title", nil, "nothing relevant"),
	}
	m := newModel(entries)
	m.deep = true
	m.input.SetValue("keyword")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 result in deep mode, got %d", len(m.filtered))
	}
	if len(m.deepContexts) != 1 {
		t.Fatalf("expected 1 deep context, got %d", len(m.deepContexts))
	}
	if m.deepContexts[0] == "" {
		t.Error("expected non-empty deep context for snippet match")
	}
}

func TestApplyFilterDeepOffNoContexts(t *testing.T) {
	entries := []storage.RecordEntry{
		entry("Architecture Record", nil, ""),
	}
	m := newModel(entries)
	m.deep = false
	m.input.SetValue("arch")
	m.applyFilter()

	if m.deepContexts != nil {
		t.Error("deepContexts should be nil when deep mode is off")
	}
}

func TestApplyFilterNoResults(t *testing.T) {
	entries := []storage.RecordEntry{
		entry("Alpha", nil, ""),
		entry("Beta", nil, ""),
	}
	m := newModel(entries)
	m.input.SetValue("zzzznotfound")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 results, got %d", len(m.filtered))
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"zero maxLen", "hello", 0, ""},
		{"negative maxLen", "hello", -1, ""},
		{"fits", "hi", 10, "hi"},
		{"exact length", "hello", 5, "hello"},
		{"truncates with ellipsis", "hello world", 8, "hello w…"},
		{"maxLen 1", "hello", 1, "h"},
		{"maxLen 3", "hello", 3, "hel"},
		{"maxLen 4 truncates", "hello", 4, "hel…"},
		{"unicode aware", "héllo wörld", 7, "héllo …"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}
