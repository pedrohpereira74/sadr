package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func setupNewTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sadrDir := filepath.Join(dir, ".sadr", "records")
	if err := os.MkdirAll(sadrDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	return dir
}

func TestNewCreatesRecord(t *testing.T) {
	dir := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "--title", "Use retry with backoff"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	entries, err := os.ReadDir(recordsDir)
	if err != nil {
		t.Fatalf("failed to read records dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 record, got %d", len(entries))
	}
}
