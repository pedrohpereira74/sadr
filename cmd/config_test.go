package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigOpensLocal(t *testing.T) {
	dir := t.TempDir()
	sadrDir := filepath.Join(dir, ".sadr")
	if err := os.MkdirAll(sadrDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	configPath := filepath.Join(sadrDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("fields: []\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	t.Setenv("EDITOR", "true")

	rootCmd.SetArgs([]string{"config"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigGlobalCreatesDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EDITOR", "true")

	rootCmd.SetArgs([]string{"config", "--global"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sadrGlobal := filepath.Join(home, ".sadr")
	if _, err := os.Stat(sadrGlobal); os.IsNotExist(err) {
		t.Error("expected ~/.sadr/ to be created")
	}
	if _, err := os.Stat(filepath.Join(sadrGlobal, "config.yaml")); os.IsNotExist(err) {
		t.Error("expected ~/.sadr/config.yaml to be created")
	}
}
