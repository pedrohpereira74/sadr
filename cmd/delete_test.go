package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func setupDeleteTest(t *testing.T) (dir string, username string) {
	t.Helper()
	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	username = setupTestUser(t, home)

	dir, _ = os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	s := storage.NewStorage(recordsDir)
	r, _ := model.NewRecordWithOptions("To be deleted", "full")
	_, _ = s.SaveRecord(r)

	originalWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })
	return dir, username
}

func TestDeleteRemovesRecord(t *testing.T) {
	dir, username := setupDeleteTest(t)

	rootCmd.SetArgs([]string{"delete", "--id", "1", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 0 {
		t.Errorf("expected 0 records after delete, got %d", len(entries))
	}
}
