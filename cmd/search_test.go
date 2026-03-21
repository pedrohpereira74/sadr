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
	dir := t.TempDir()
	recordsDir := filepath.Join(dir, ".sadr", "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	s := storage.NewStorage(recordsDir)
	r, _ := model.NewRecord("Use retry with backoff")
	_, _ = s.SaveRecord(r)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

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
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"search", "banana"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected 'no results' message, got empty")
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
