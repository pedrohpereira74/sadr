package cmd

import (
	"os"
	"path/filepath"
	"strings"
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

func TestConfigSetAPIKeyCreatesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	rootCmd.SetArgs([]string{"config", "--set-api-key", "test-key-123"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configPath := filepath.Join(home, ".sadr", "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if !strings.Contains(string(content), `api_key: "test-key-123"`) {
		t.Errorf("expected api key in config, got:\n%s", string(content))
	}
}

func TestConfigSetAPIKeyUpdatesExistingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	globalDir := filepath.Join(home, ".sadr")
	_ = os.MkdirAll(globalDir, 0755)
	configPath := filepath.Join(globalDir, "config.yaml")

	initialContent := "editor: \"nano\"\napi_key: \"old-key\"\nsome_other_setting: true\n"
	_ = os.WriteFile(configPath, []byte(initialContent), 0644)

	rootCmd.SetArgs([]string{"config", "--set-api-key", "new-key-456"})
	_ = rootCmd.Execute()

	content, _ := os.ReadFile(configPath)
	strContent := string(content)

	if !strings.Contains(strContent, `api_key: "new-key-456"`) {
		t.Errorf("expected updated api key, got:\n%s", strContent)
	}
	if strings.Contains(strContent, "old-key") {
		t.Errorf("did not expect old key, got:\n%s", strContent)
	}
	if !strings.Contains(strContent, `editor: "nano"`) {
		t.Errorf("expected original content to be preserved, got:\n%s", strContent)
	}
}
