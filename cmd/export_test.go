package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func setupExportTest(t *testing.T) string {
	t.Helper()
	exportID = 0
	exportAll = false
	exportTags = ""

	dir := t.TempDir()
	recordsDir := filepath.Join(dir, ".sadr", "records")
	exportsDir := filepath.Join(dir, ".sadr", "exports")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatalf("failed to create records dir: %v", err)
	}
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		t.Fatalf("failed to create exports dir: %v", err)
	}

	s := storage.NewStorage(recordsDir)
	r1, _ := model.NewRecord("Use retry with backoff")
	r1.Snippet = "client := retryablehttp.NewClient()"
	r1.Fields["tags"] = "api,performance"
	_, _ = s.SaveRecord(r1)

	r2, _ := model.NewRecord("Redis cache strategy")
	r2.Fields["tags"] = "database,performance"
	_, _ = s.SaveRecord(r2)

	r3, _ := model.NewRecord("Auth token rotation")
	r3.Fields["tags"] = "security,api"
	_, _ = s.SaveRecord(r3)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	return dir
}

func TestExportCreatesHTMLFile(t *testing.T) {
	dir := setupExportTest(t)

	rootCmd.SetArgs([]string{"export", "--id", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exportsDir := filepath.Join(dir, ".sadr", "exports")
	entries, _ := os.ReadDir(exportsDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 export, got %d", len(entries))
	}
}

func TestExportAllCreatesMultipleFiles(t *testing.T) {
	dir := setupExportTest(t)

	rootCmd.SetArgs([]string{"export", "--all"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exportsDir := filepath.Join(dir, ".sadr", "exports")
	entries, _ := os.ReadDir(exportsDir)
	if len(entries) != 3 {
		t.Errorf("expected 3 exports, got %d", len(entries))
	}
}

func TestExportFiltersByTags(t *testing.T) {
	dir := setupExportTest(t)

	rootCmd.SetArgs([]string{"export", "--tags", "security"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exportsDir := filepath.Join(dir, ".sadr", "exports")
	entries, _ := os.ReadDir(exportsDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 export with tag 'security', got %d", len(entries))
	}
}
