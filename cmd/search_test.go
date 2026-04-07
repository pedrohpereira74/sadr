package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func setupSearchTest(t *testing.T) string {
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
	r, _ := model.NewRecordWithOptions("Use retry with backoff", "full")
	r.Author = username
	_, _ = s.SaveRecord(r)

	originalWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })

	return dir
}

func TestSearchFindsRecord(t *testing.T) {
	setupSearchTest(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"search", "retry"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected output, got empty")
	}
}

func TestSearchNoResults(t *testing.T) {
	setupSearchTest(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"search", "banana"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if len(output) != 0 {
		t.Errorf("expected no stdout output for zero results, got: %s", output)
	}
}

func TestSearchByID(t *testing.T) {
	setupSearchTest(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"search", "--id", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected full record output, got empty")
	}
}
