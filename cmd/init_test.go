package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesSadrDirectory(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	originalWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })

	rootCmd.SetArgs([]string{"init", "--preset", "minimal"})
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

func TestInitMinimalCreatesConfig(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--preset", "minimal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".sadr", "config.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if !strings.Contains(string(content), "name: title") {
		t.Error("expected config to contain title field")
	}
	if !strings.Contains(string(content), "name: tags") {
		t.Error("expected config to contain tags field")
	}
	if strings.Contains(string(content), "name: status") {
		t.Error("minimal config should not contain status field")
	}
}

func TestInitExtendedCreatesConfig(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--preset", "extended"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".sadr", "config.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if !strings.Contains(string(content), "name: status") {
		t.Error("extended config should contain status field")
	}
	if !strings.Contains(string(content), "name: context") {
		t.Error("extended config should contain context field")
	}
}

func TestInitDoesNotOverwrite(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	sadrDir := filepath.Join(dir, ".sadr")
	if err := os.MkdirAll(sadrDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--preset", "minimal"})
	_ = rootCmd.Execute()
}

func TestInitAddsExportsToGitignore(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--preset", "minimal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to exist: %v", err)
	}
	if !strings.Contains(string(content), ".sadr/exports/") {
		t.Error("expected .gitignore to contain .sadr/exports/")
	}
}

func TestInitDoesNotDuplicateGitignore(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	existing := "node_modules/\n.sadr/exports/\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0644); err != nil {
		t.Fatalf("failed to write gitignore: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--preset", "minimal"})
	_ = rootCmd.Execute()

	content, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	count := strings.Count(string(content), ".sadr/exports/")
	if count != 1 {
		t.Errorf("expected 1 occurrence of .sadr/exports/, got %d", count)
	}
}
