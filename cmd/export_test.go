package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func setupExportTest(t *testing.T) (dir string, username string) {
	t.Helper()
	resetCmd(findSubCmd("export"))

	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	username = setupTestUser(t, home)

	dir, _ = os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatalf("failed to create records dir: %v", err)
	}

	s := storage.NewStorage(recordsDir)
	r1, _ := model.NewRecordWithOptions("Use retry with backoff", "full")
	r1.Snippet = "client := retryablehttp.NewClient()"
	r1.Fields["tags"] = "api,performance"
	_, _ = s.SaveRecord(r1)

	r2, _ := model.NewRecordWithOptions("Redis cache strategy", "full")
	r2.Fields["tags"] = "database,performance"
	_, _ = s.SaveRecord(r2)

	r3, _ := model.NewRecordWithOptions("Auth token rotation", "full")
	r3.Fields["tags"] = "security,api"
	_, _ = s.SaveRecord(r3)

	originalWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })
	return dir, username
}

func TestExportCreatesHTMLFile(t *testing.T) {
	dir, username := setupExportTest(t)

	rootCmd.SetArgs([]string{"export", "--id", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exportsDir := filepath.Join(dir, ".sadr", username, "exports")
	entries, _ := os.ReadDir(exportsDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 export, got %d", len(entries))
	}
}

func TestExportAllCreatesMultipleFiles(t *testing.T) {
	dir, username := setupExportTest(t)

	rootCmd.SetArgs([]string{"export", "--all"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exportsDir := filepath.Join(dir, ".sadr", username, "exports")
	entries, _ := os.ReadDir(exportsDir)
	if len(entries) != 3 {
		t.Errorf("expected 3 exports, got %d", len(entries))
	}
}

func TestExportFiltersByTags(t *testing.T) {
	dir, username := setupExportTest(t)

	rootCmd.SetArgs([]string{"export", "--tags", "security"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exportsDir := filepath.Join(dir, ".sadr", username, "exports")
	entries, _ := os.ReadDir(exportsDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 export with tag 'security', got %d", len(entries))
	}
}
