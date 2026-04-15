package model

import (
	"testing"
	"time"
)

func TestNewRecordHasTitle(t *testing.T) {
	r, err := NewRecordWithOptions("Use retry with exponential backoff", "full")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.Title != "Use retry with exponential backoff" {
		t.Errorf("expected 'Use retry with exponential backoff, got '%s'", r.Title)
	}
}

func TestNewRecordWithInvalidTitleReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"empty", ""},
		{"whitespace only", "  "},
		{"tabs only", "\t\t"},
	}

	for _, tt := range tests {
		_, err := NewRecordWithOptions(tt.title, "full")
		if err == nil {
			t.Errorf("expected error for title %q, got nil", tt.title)
		}
	}
}

func TestNewRecordHasCreatedAt(t *testing.T) {
	before := time.Now()
	r, err := NewRecordWithOptions("Use retry with exponential backoff", "full")
	after := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.CreatedAt.Before(before) {
		t.Errorf("expected createdAt between %v and %v, got %v", before, after, r.CreatedAt)
	}
}

func TestNewRecordDefaultTypeIsFull(t *testing.T) {
	r, err := NewRecordWithOptions("Use retry with exponential backoff", "full")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Type != "full" {
		t.Errorf("expected type 'full', got '%s'", r.Type)
	}
}

func TestNewRecordWithTags(t *testing.T) {
	_, err := NewRecordWithOptions("Use retry", "full")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRecordWithInvalidTypeReturnsError(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
	}{
		{"empty type", ""},
		{"unknown type", "banana"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRecordWithOptions("Use retry", tt.recordType)

			if err == nil {
				t.Errorf("expected error for type %q, got nil", tt.recordType)
			}
		})
	}
}

func TestNewRecordWithSnippet(t *testing.T) {
	r, err := NewRecordWithOptions("Use retry", "full")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Snippet != "" {
		t.Errorf("expected empty snippet by default, got '%s'", r.Snippet)
	}
}

func TestNewRecordFieldsIsInitialized(t *testing.T) {
	r, err := NewRecordWithOptions("Use retry", "full")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Fields == nil {
		t.Error("expected Fields to be initialized, got nil")
	}
}

func TestRecordCanSetAndGetFields(t *testing.T) {
	r, err := NewRecordWithOptions("Use retry", "full")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r.Fields["language"] = "go"
	r.Status = "accepted"
	r.Tags = []string{"api", "performance"}

	if r.Fields["language"] != "go" {
		t.Errorf("expected 'go', got '%s'", r.Fields["language"])
	}
	if r.Status != "accepted" {
		t.Errorf("expected 'accepted', got '%s'", r.Status)
	}
	if len(r.Tags) != 2 || r.Tags[0] != "api" {
		t.Errorf("expected tags [api performance], got %v", r.Tags)
	}
}

func TestNewRecordHasSchemaVersion(t *testing.T) {
	r, err := NewRecordWithOptions("Use retry", "full")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %d, got %d", SchemaVersion, r.SchemaVersion)
	}
}

func TestNewRecordHasDefaultFileRef(t *testing.T) {
	r, err := NewRecordWithOptions("Use retry", "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.FileRef != "N/A" {
		t.Errorf("expected file_ref 'N/A', got '%s'", r.FileRef)
	}
}
