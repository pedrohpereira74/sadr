package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func setupDeleteTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	recordsDir := filepath.Join(dir, ".sadr", "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	s := storage.NewStorage(recordsDir)
	r, _ := model.NewRecord("To be deleted")
	_, _ = s.SaveRecord(r)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	return dir
}

func TestDeleteRemovesRecord(t *testing.T) {
	dir := setupDeleteTest(t)

	rootCmd.SetArgs([]string{"delete", "--id", "1", "--confirm"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 0 {
		t.Errorf("expected 0 records after delete, got %d", len(entries))
	}
}
