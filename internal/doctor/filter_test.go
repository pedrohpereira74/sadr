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
