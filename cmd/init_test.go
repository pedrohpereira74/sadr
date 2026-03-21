package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesSadrDirectory(t *testing.T) {
	dir := t.TempDir()
	err := os.Chdir(dir)
	if err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	rootCmd.SetArgs([]string{"init"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".sadr")); os.IsNotExist(err) {
		t.Error("expected .sadr/ to exist")
	}
	if _, err := os.Stat(filepath.Join(dir, ".sadr", "records")); os.IsNotExist(err) {
		t.Error("expected .sadr/records/ to exist")
	}
	if _, err := os.Stat(filepath.Join(dir, ".sadr", "exports")); os.IsNotExist(err) {
		t.Error("expected .sadr/exports/ to exist")
	}
}
