package search

import (
	"strings"
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

// --- MatchesTags ---

func TestMatchesTagsSingleTag(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Tags = []string{"golang"}
	if !MatchesTags(r, "golang") {
		t.Error("expected tag match")
	}
}

func TestMatchesTagsOneOfMany(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Tags = []string{"golang", "backend", "api"}
	if !MatchesTags(r, "backend") {
		t.Error("expected match against one of multiple tags")
	}
}

func TestMatchesTagsCaseInsensitive(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Tags = []string{"Golang"}
	if !MatchesTags(r, "GOLANG") {
		t.Error("expected case-insensitive tag match")
	}
}

func TestMatchesTagsNoMatch(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Tags = []string{"golang"}
	if MatchesTags(r, "python") {
		t.Error("expected no tag match")
	}
}

func TestMatchesTagsNilTags(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	if MatchesTags(r, "anything") {
		t.Error("expected no match on nil tags")
	}
}

// --- MatchesDeep ---

func TestMatchesDeepSnippet(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "uses dependency injection pattern"
	if !MatchesDeep(r, "injection") {
		t.Error("expected snippet match")
	}
}

func TestMatchesDeepField(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Fields = map[string]string{"context": "uses event sourcing"}
	if !MatchesDeep(r, "sourcing") {
		t.Error("expected field match")
	}
}

func TestMatchesDeepCaseInsensitive(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "Uses Dependency Injection"
	if !MatchesDeep(r, "dependency") {
		t.Error("expected case-insensitive snippet match")
	}
}

func TestMatchesDeepNoMatch(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "something completely different"
	r.Fields = map[string]string{"k": "v"}
	if MatchesDeep(r, "zzz") {
		t.Error("expected no deep match")
	}
}

// --- DeepContext ---

func TestDeepContextEmptyRecord(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	if got := DeepContext(r, "anything"); got != "" {
		t.Errorf("expected empty context, got %q", got)
	}
}

func TestDeepContextNoMatch(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "completely different text"
	if got := DeepContext(r, "zzz"); got != "" {
		t.Errorf("expected empty context for no match, got %q", got)
	}
}

func TestDeepContextSnippetMatchContainsWord(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "contains the keyword right here in the snippet"
	got := DeepContext(r, "keyword")
	if got == "" {
		t.Fatal("expected non-empty context")
	}
	if !strings.Contains(got, "keyword") {
		t.Errorf("context should contain the matched word, got %q", got)
	}
}

func TestDeepContextPrefersSnippetOverFields(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "snippet has needle"
	r.Fields = map[string]string{"ctx": "field also has needle"}
	r.FieldOrder = []string{"ctx"}
	got := DeepContext(r, "needle")
	if !strings.Contains(got, "snippet") {
		t.Errorf("expected snippet context, got %q", got)
	}
}

func TestDeepContextFallsBackToField(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "no match here"
	r.Fields = map[string]string{"ctx": "field contains needle"}
	r.FieldOrder = []string{"ctx"}
	got := DeepContext(r, "needle")
	if got == "" {
		t.Fatal("expected context from field")
	}
	if !strings.Contains(got, "needle") {
		t.Errorf("context should contain matched word, got %q", got)
	}
}

func TestDeepContextRespectsFieldOrder(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Fields = map[string]string{
		"first":  "first field has needle",
		"second": "second field has needle",
	}
	r.FieldOrder = []string{"first", "second"}
	got := DeepContext(r, "needle")
	if !strings.Contains(got, "first") {
		t.Errorf("expected context from first field in FieldOrder, got %q", got)
	}
}

func TestDeepContextLeadingDotsForMidTextMatch(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "long prefix text before the keyword appears in this sentence"
	got := DeepContext(r, "keyword")
	if !strings.HasPrefix(got, "...") {
		t.Errorf("expected leading '...' for mid-text match, got %q", got)
	}
}

func TestDeepContextNoLeadingDotsForStartMatch(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "keyword is right at the start of this text"
	got := DeepContext(r, "keyword")
	if strings.HasPrefix(got, "...") {
		t.Errorf("expected no leading '...' for start match, got %q", got)
	}
}

func TestDeepContextTrailingDotsWhenTextContinues(t *testing.T) {
	r, _ := model.NewRecordWithOptions("Title", "full")
	r.Snippet = "keyword appears early and there is much more text after it here to exceed the window"
	got := DeepContext(r, "keyword")
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected trailing '...' when text continues past window, got %q", got)
	}
}
