package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/admin"
)

func TestConfigOpensLocal(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	configsDir := filepath.Join(dir, ".sadr", "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	configPath := filepath.Join(configsDir, "default-config.yaml")
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

func TestConfigSetAPIKeyCancelledOnOverwrite(t *testing.T) {
	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	globalDir := filepath.Join(home, ".sadr")
	_ = os.MkdirAll(globalDir, 0755)
	configPath := filepath.Join(globalDir, "global-config.yaml")
	_ = os.WriteFile(configPath, []byte("api_key: \"existing-key\"\n"), 0644)

	old := confirmOverwrite
	confirmOverwrite = func() string { return "no" }
	t.Cleanup(func() { confirmOverwrite = old })

	resetCmd(findSubCmd("config"))
	rootCmd.SetArgs([]string{"config", "--set-api-key", "new-key"})
	_ = rootCmd.Execute()

	content, _ := os.ReadFile(configPath)
	if !strings.Contains(string(content), "existing-key") {
		t.Error("expected existing key to be preserved when user cancels")
	}
	if strings.Contains(string(content), "new-key") {
		t.Error("expected new key NOT to be written when user cancels")
	}
}

func TestConfigSetAPIKeyNoPromptWhenEmpty(t *testing.T) {
	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	globalDir := filepath.Join(home, ".sadr")
	_ = os.MkdirAll(globalDir, 0755)
	configPath := filepath.Join(globalDir, "global-config.yaml")
	_ = os.WriteFile(configPath, []byte("api_key: \"\"\n"), 0644)

	prompted := false
	old := confirmOverwrite
	confirmOverwrite = func() string { prompted = true; return "no" }
	t.Cleanup(func() { confirmOverwrite = old })

	resetCmd(findSubCmd("config"))
	rootCmd.SetArgs([]string{"config", "--set-api-key", "fresh-key"})
	_ = rootCmd.Execute()

	if prompted {
		t.Error("should not prompt when existing key is empty")
	}

	content, _ := os.ReadFile(configPath)
	if !strings.Contains(string(content), "fresh-key") {
		t.Error("expected new key to be written without prompting")
	}
}

func TestSetupAdminCreatesHash(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })

	sadrRoot := filepath.Join(dir, ".sadr")
	_ = os.MkdirAll(filepath.Join(sadrRoot, "configs"), 0755)
	_ = os.WriteFile(filepath.Join(sadrRoot, "configs", "default-config.yaml"), []byte("fields: []\n"), 0644)

	originalWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(originalWd) })

	resetCmd(findSubCmd("config"))
	rootCmd.SetArgs([]string{"config", "--setup-admin"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !admin.IsConfigured(sadrRoot) {
		t.Error("expected admin to be configured after --setup-admin")
	}
}

func TestSetupAdminRegeneratesToken(t *testing.T) {
	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })

	sadrRoot := filepath.Join(dir, ".sadr")
	_ = os.MkdirAll(filepath.Join(sadrRoot, "configs"), 0755)
	_ = os.WriteFile(filepath.Join(sadrRoot, "configs", "default-config.yaml"), []byte("fields: []\n"), 0644)

	token1, _ := admin.Setup(sadrRoot)

	originalWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(originalWd) })

	resetCmd(findSubCmd("config"))
	rootCmd.SetArgs([]string{"config", "--setup-admin"})
	_ = rootCmd.Execute()

	t.Setenv("SADR_ADMIN_TOKEN", token1)
	if err := admin.RequireAdmin(sadrRoot); err == nil {
		t.Error("expected old token to be invalid after regeneration")
	}
}
