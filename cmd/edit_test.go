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
	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	username := setupTestUser(t, home)

	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	s := storage.NewStorage(recordsDir)
	r, _ := model.NewRecordWithOptions("Editable record", "full")
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
	_ = setupEditTest(t)

	t.Setenv("EDITOR", "sort")

	rootCmd.SetArgs([]string{"edit", "--id", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
