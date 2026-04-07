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
	if _, err := os.Stat(filepath.Join(dir, ".sadr", "configs", "default-config.yaml")); os.IsNotExist(err) {
		t.Error("expected .sadr/configs/default-config.yaml to exist")
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

	content, err := os.ReadFile(filepath.Join(dir, ".sadr", "configs", "default-config.yaml"))
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

	content, err := os.ReadFile(filepath.Join(dir, ".sadr", "configs", "default-config.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if !strings.Contains(string(content), "name: context") {
		t.Error("extended config should contain context field")
	}
}

func TestInitDoesNotOverwrite(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	sadrDir := filepath.Join(dir, ".sadr")
	configsDir := filepath.Join(sadrDir, "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defaultConfigPath := filepath.Join(configsDir, "default-config.yaml")
	if err := os.WriteFile(defaultConfigPath, []byte("fields: []"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--preset", "minimal"})
	_ = rootCmd.Execute()

	content, _ := os.ReadFile(defaultConfigPath)
	if string(content) != "fields: []" {
		t.Error("default-config.yaml was overwritten when it should have been preserved")
	}
}

func TestInitAddsExportsToGitignore(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	_ = os.Mkdir(filepath.Join(dir, ".git"), 0755)
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
	if !strings.Contains(string(content), ".sadr/*/exports/") {
		t.Error("expected .gitignore to contain .sadr/*/exports/")
	}
	if !strings.Contains(string(content), ".sadr/*/answers/") {
		t.Error("expected .gitignore to contain .sadr/*/answers/")
	}
}

func TestInitDoesNotDuplicateGitignore(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	_ = os.Mkdir(filepath.Join(dir, ".git"), 0755)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	existing := "node_modules/\n.sadr/*/exports/\n.sadr/*/answers/\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0644); err != nil {
		t.Fatalf("failed to write gitignore: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--preset", "minimal"})
	_ = rootCmd.Execute()

	content, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	count := strings.Count(string(content), ".sadr/*/exports/")
	if count != 1 {
		t.Errorf("expected 1 occurrence of .sadr/*/exports/, got %d", count)
	}
}

func TestInitHealsWhenConfigMissing(t *testing.T) {
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
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaultConfigPath := filepath.Join(sadrDir, "configs", "default-config.yaml")
	if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
		t.Error("expected configs/default-config.yaml to be recreated by healing")
	}
}
