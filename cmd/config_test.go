package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigOpensLocal(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	sadrDir := filepath.Join(dir, ".sadr")
	if err := os.MkdirAll(sadrDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	configPath := filepath.Join(sadrDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("fields: []\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	originalWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })

	t.Setenv("EDITOR", "sort")

	resetCmd(findSubCmd("config"))
	rootCmd.SetArgs([]string{"config"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigGlobalDoesNotCreateDirectory(t *testing.T) {
	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("EDITOR", "sort")

	resetCmd(findSubCmd("config"))
	rootCmd.SetArgs([]string{"config", "--global"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sadrGlobal := filepath.Join(home, ".sadr")
	if _, err := os.Stat(sadrGlobal); !os.IsNotExist(err) {
		t.Error("expected ~/.sadr/ NOT to be created")
	}
}

func TestConfigSetAPIKeyCreatesFile(t *testing.T) {
	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	resetCmd(findSubCmd("config"))
	rootCmd.SetArgs([]string{"config", "--set-api-key", "test-key-123"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configPath := filepath.Join(home, ".sadr", "global-config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if !strings.Contains(string(content), `api_key: "test-key-123"`) {
		t.Errorf("expected api key in config, got:\n%s", string(content))
	}
}

func TestConfigSetAPIKeyUpdatesExistingFile(t *testing.T) {
	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	globalDir := filepath.Join(home, ".sadr")
	_ = os.MkdirAll(globalDir, 0755)
	configPath := filepath.Join(globalDir, "global-config.yaml")

	initialContent := "editor: \"nano\"\napi_key: \"old-key\"\nsome_other_setting: true\n"
	_ = os.WriteFile(configPath, []byte(initialContent), 0644)

	resetCmd(findSubCmd("config"))
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
