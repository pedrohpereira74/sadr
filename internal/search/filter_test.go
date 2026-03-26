package search

import "testing"

func TestHasAnyTagFindsMatch(t *testing.T) {
	if !HasAnyTag("api,security", "api") {
		t.Error("expected match for 'api'")
	}
}

func TestHasAnyTagNoMatch(t *testing.T) {
	if HasAnyTag("api,security", "database") {
		t.Error("expected no match for 'database'")
	}
}

func TestHasAnyTagMultipleFilters(t *testing.T) {
	if !HasAnyTag("api,security", "database,security") {
		t.Error("expected match for 'security'")
	}
}

func TestHasAnyTagHandlesSpaces(t *testing.T) {
	if !HasAnyTag("api, security", "security") {
		t.Error("expected match even with spaces")
	}
}

func TestHasAnyTagCaseInsensitive(t *testing.T) {
	if !HasAnyTag("API,Security", "api") {
		t.Error("expected case-insensitive match for 'API' vs 'api'")
	}
	if !HasAnyTag("database", "DATABASE") {
		t.Error("expected case-insensitive match for 'database' vs 'DATABASE'")
	}
	if !HasAnyTag("Api, Security", "security,tooling") {
		t.Error("expected case-insensitive match for mixed case tags")
	}
}
