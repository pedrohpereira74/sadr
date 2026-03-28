package cmd

import (
	"os"
	"testing"

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

	ui.PauseFn = func(_ float64) {}

	os.Exit(m.Run())
}

// resetCmd resets all flags on a command to their default values and clears Changed state.
// This is needed when rootCmd.Execute() is called multiple times in the same test binary.
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

// findSubCmd finds a direct subcommand of rootCmd by name.
func findSubCmd(name string) *cobra.Command {
	for _, c := range rootCmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
