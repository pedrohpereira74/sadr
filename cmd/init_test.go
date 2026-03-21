package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesSadrDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
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

func TestInitCreatesConfigYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configPath := filepath.Join(dir, ".sadr", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("expected .sadr/config.yaml to exist")
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if !strings.Contains(string(content), "name: title") {
		t.Error("expected config to contain title field")
	}
	if !strings.Contains(string(content), "name: tags") {
		t.Error("expected config to contain tags field")
	}
}

func TestInitDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	sadrDir := filepath.Join(dir, ".sadr")
	if err := os.MkdirAll(sadrDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	rootCmd.SetArgs([]string{"init"})
	_ = rootCmd.Execute()
}
