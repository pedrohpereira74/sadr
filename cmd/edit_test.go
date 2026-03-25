package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func setupEditTest(t *testing.T) string {
	t.Helper()
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	recordsDir := filepath.Join(dir, ".sadr", "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	s := storage.NewStorage(recordsDir)
	r, _ := model.NewRecord("Editable record")
	r.Fields["status"] = "proposed"
	_, _ = s.SaveRecord(r)

	originalWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })
	return dir
}

func TestEditFindsRecord(t *testing.T) {
	dir := setupEditTest(t)

	os.Setenv("EDITOR", "sort")
	defer os.Unsetenv("EDITOR")

	rootCmd.SetArgs([]string{"edit", "--id", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 record, got %d", len(entries))
	}
}
