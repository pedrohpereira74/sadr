package search

import "testing"

func TestHasAnyTagFindsMatch(t *testing.T) {
	if !HasAnyTag([]string{"api", "security"}, "api") {
		t.Error("expected match for 'api'")
	}
}

func TestHasAnyTagNoMatch(t *testing.T) {
	if HasAnyTag([]string{"api", "security"}, "database") {
		t.Error("expected no match for 'database'")
	}
}

func TestHasAnyTagMultipleFilters(t *testing.T) {
	if !HasAnyTag([]string{"api", "security"}, "database,security") {
		t.Error("expected match for 'security'")
	}
}

func TestHasAnyTagEmptyRecord(t *testing.T) {
	if HasAnyTag([]string{}, "api") {
		t.Error("expected no match for empty record tags")
	}
}

func TestHasAnyTagCaseInsensitive(t *testing.T) {
	if !HasAnyTag([]string{"API", "Security"}, "api") {
		t.Error("expected case-insensitive match for 'API' vs 'api'")
	}
	if !HasAnyTag([]string{"database"}, "DATABASE") {
		t.Error("expected case-insensitive match for 'database' vs 'DATABASE'")
	}
	if !HasAnyTag([]string{"Api", "Security"}, "security,tooling") {
		t.Error("expected case-insensitive match for mixed case tags")
	}
}
