package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func setupListTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	recordsDir := filepath.Join(dir, ".sadr", "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	s := storage.NewStorage(recordsDir)

	r1, _ := model.NewRecordWithOptions("Auth retry logic", "snippet")
	r1.Fields["tags"] = "api,security"
	_, _ = s.SaveRecord(r1)

	r2, _ := model.NewRecordWithOptions("Use Redis for cache", "adr")
	r2.Fields["tags"] = "database,performance"
	_, _ = s.SaveRecord(r2)

	r3, _ := model.NewRecordWithOptions("Error handling pattern", "snippet")
	r3.Fields["tags"] = "api,architecture"
	_, _ = s.SaveRecord(r3)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	return dir
}

func TestListShowsAllRecords(t *testing.T) {
	setupListTest(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 records, got %d", len(lines))
	}
}

func TestListFiltersByType(t *testing.T) {
	setupListTest(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--type", "snippet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 snippets, got %d", len(lines))
	}
}

func TestListFiltersByTags(t *testing.T) {
	setupListTest(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list", "--tags", "api"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 records with tag 'api', got %d", len(lines))
	}
}

func TestListFiltersByField(t *testing.T) {
	setupListTest(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	listType = ""
	listTags = ""
	rootCmd.SetArgs([]string{"list", "--field", "tags=database,performance"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 record, got %d", len(lines))
	}
}
