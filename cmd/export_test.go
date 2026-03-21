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
	r, _ := model.NewRecord("Use retry with backoff")
	r.Snippet = "client := retryablehttp.NewClient()"
	r.Fields["status"] = "accepted"
	_, _ = s.SaveRecord(r)

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
