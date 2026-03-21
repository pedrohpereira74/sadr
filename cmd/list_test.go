package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func TestListShowsRecords(t *testing.T) {
	dir := t.TempDir()
	recordsDir := filepath.Join(dir, ".sadr", "records")
	if err := os.MkdirAll(recordsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	s := storage.NewStorage(recordsDir)
	r1, _ := model.NewRecord("First record")
	r2, _ := model.NewRecord("Second record")
	_, _ = s.SaveRecord(r1)
	_, _ = s.SaveRecord(r2)

	err := os.Chdir(dir)
	if err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"list"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected output, got empty")
	}
}
