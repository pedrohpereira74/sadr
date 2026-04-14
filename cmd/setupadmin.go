package cmd

import (
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/admin"
	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/ui"
)

func runSetupAdmin() {
	dir, err := os.Getwd()
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("could not get working directory: %v", err))
		return
	}
	paths, err := discover.FindSadrDir(dir)
	if err != nil {
		ui.Error(os.Stderr, "no sadr project found — run 'sadr init' first")
		return
	}

	if admin.IsConfigured(paths.Root) {
		if !confirmPromptFn("admin already configured. regenerate token?") {
			return
		}
	}

	token, err := admin.Setup(paths.Root)
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to configure admin: %v", err))
		return
	}

	fmt.Fprintln(os.Stderr)
	ui.Success(os.Stderr, "admin configured.")
	fmt.Fprintln(os.Stderr)
	ui.Info(os.Stderr, "generated token (store it somewhere safe — it won't be shown again):")
	fmt.Fprintf(os.Stderr, "\n  SADR_ADMIN_TOKEN=%s\n\n", token)
	ui.Info(os.Stderr, "add it to your ~/.zshrc or ~/.bashrc and never share it.")
	ui.Info(os.Stderr, "the hash was saved to .sadr/admin.yaml and is safe to commit.")
}
