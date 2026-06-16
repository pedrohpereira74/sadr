package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/pedrohpereira74/sadr/internal/wizard"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestMain(m *testing.M) {
	editorRunner = func(_ string, _ string) error { return nil }
	wizardRunner = func(opts wizard.Options) (map[string]string, error) {
		if opts.Suggestions != nil {
			return opts.Suggestions, nil
		}
		return map[string]string{"title": "Mock Title"}, nil
	}
	presetSelector = func() string { return "minimal" }
	fallbackPrompter = func() string { return "yes" }
	snippetCapturer = func() (string, error) { return "mock snippet", nil }
	clipboardReader = func() (string, error) { return "mock clipboard", nil }
	confirmOverwrite = func() string { return "yes" }
	confirmPromptFn = func(_ string) bool { return true }
	generateTextFn = func(_ context.Context, _, _, _, _ string, _ time.Duration) (string, error) {
		return "# Mock Report\n\nThis is a mock AI response.", nil
	}

	ui.PauseFn = func(_ float64) {}

	os.Exit(m.Run())
}

func resetCmd(cmd *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if f.Changed {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		}
	}
	cmd.Flags().VisitAll(reset)
	cmd.PersistentFlags().VisitAll(reset)
}

func findSubCmd(name string) *cobra.Command {
	for _, c := range rootCmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func setupTestUser(t *testing.T, home string) string {
	t.Helper()
	const testUser = "testuser"
	sadrHome := filepath.Join(home, ".sadr")
	if err := os.MkdirAll(sadrHome, 0755); err != nil {
		t.Fatalf("failed to create ~/.sadr: %v", err)
	}
	globalConfig := `username: "testuser"
editor: ""
language: "english"
ai:
  provider: "gemini"
  api_key: ""
  model: ""
  ai_depth: false
`
	if err := os.WriteFile(filepath.Join(sadrHome, "global-config.yaml"), []byte(globalConfig), 0600); err != nil {
		t.Fatalf("failed to write global config: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return testUser
}
