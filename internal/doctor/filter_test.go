package doctor

import (
	"reflect"
	"testing"
)

func TestFilterChangedFiles(t *testing.T) {
	records := []RecordRef{
		{ID: "alice/1", FileRef: "api/server.go", Status: "active"},
		{ID: "bob/2", FileRef: "api/server.go", Status: "active"}, // same file, two records
		{ID: "alice/3", FileRef: "db/store.go", Status: "active"},
		{ID: "alice/4", FileRef: "old.go", Status: "proposed"}, // not active
	}
	changed := []string{"api/server.go", "README.md", "db/store.go", "api/server.go"}

	got := FilterChangedFiles(changed, records)
	want := []AuditTarget{
		{FileRef: "api/server.go", Records: []string{"alice/1", "bob/2"}},
		{FileRef: "db/store.go", Records: []string{"alice/3"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterChangedFiles = %+v, want %+v", got, want)
	}
}

func TestFilterChangedFilesNoMatches(t *testing.T) {
	records := []RecordRef{{ID: "a/1", FileRef: "x.go", Status: "active"}}
	if got := FilterChangedFiles([]string{"y.go", "z.go"}, records); len(got) != 0 {
		t.Errorf("expected no targets, got %+v", got)
	}
}

func TestConflicts(t *testing.T) {
	records := []RecordRef{
		{ID: "alice/1", FileRef: "api.go", Status: "active"},
		{ID: "bob/2", FileRef: "api.go", Status: "active"},   // conflict with alice/1
		{ID: "alice/3", FileRef: "solo.go", Status: "active"}, // only one record -> not a conflict
	}
	changed := []string{"api.go", "solo.go", "untracked.go"}

	got := Conflicts(changed, records)
	if len(got) != 1 || got[0].FileRef != "api.go" || len(got[0].Records) != 2 {
		t.Fatalf("expected one conflict on api.go with 2 records, got %+v", got)
	}
}

func TestRemainingConflicts(t *testing.T) {
	conflicts := []AuditTarget{
		{FileRef: "api.go", Records: []string{"alice/1", "bob/2"}},
		{FileRef: "db.go", Records: []string{"alice/3", "alice/4", "alice/5"}},
	}
	// Deprecating bob/2 resolves api.go; deprecating one of three on db.go leaves a conflict.
	remaining := RemainingConflicts(conflicts, map[string]bool{"bob/2": true, "alice/3": true})
	if len(remaining) != 1 || remaining[0].FileRef != "db.go" || len(remaining[0].Records) != 2 {
		t.Errorf("expected db.go still conflicting with 2 records, got %+v", remaining)
	}
}
