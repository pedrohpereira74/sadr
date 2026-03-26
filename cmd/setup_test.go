package cmd

import (
	"os"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/wizard"
)

func TestMain(m *testing.M) {
	// Set default mocks for all tests in the cmd package to prevent hangs
	editorRunner = func(editor, path string) {}
	wizardRunner = func(opts wizard.Options) (map[string]string, error) {
		// Default mock: return suggestions if they exist, or empty
		if opts.Suggestions != nil {
			return opts.Suggestions, nil
		}
		return map[string]string{"title": "Mock Title"}, nil
	}
	presetSelector = func() string { return "minimal" }
	fallbackPrompter = func() string { return "yes" }
	snippetCapturer = func() (string, error) { return "mock snippet", nil }

	// Ensure SADR_TEST is set for any other internal checks
	os.Setenv("SADR_TEST", "1")

	os.Exit(m.Run())
}
