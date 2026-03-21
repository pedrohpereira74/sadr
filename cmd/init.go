package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/templates"
	"github.com/spf13/cobra"
)

var initPreset string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a .sadr/ repository in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := os.Getwd()
		sadrDir := filepath.Join(dir, ".sadr")

		if _, err := os.Stat(sadrDir); !os.IsNotExist(err) {
			_, _ = fmt.Fprintln(os.Stderr,
				":(  Nice try... sadr already lives here.\n    Whats next? 'git init' inside a git repo?")
			return
		}

		if err := os.MkdirAll(filepath.Join(sadrDir, "records"), 0755); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create .sadr/records/: %v\n", err)
			return
		}
		if err := os.MkdirAll(filepath.Join(sadrDir, "exports"), 0755); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create .sadr/exports/: %v\n", err)
			return
		}

		preset := templates.MinimalConfig
		presetName := "minimal"
		if initPreset == "extended" {
			preset = templates.ExtendedConfig
			presetName = "extended"
		}

		configPath := filepath.Join(sadrDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(preset), 0644); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create config: %v\n", err)
			return
		}

		addToGitignore(dir)

		_, _ = fmt.Fprintln(os.Stderr, ":(  sadr: therapy for snippets that lost their meaning.")
		_, _ = fmt.Fprintln(os.Stderr, "")
		_, _ = fmt.Fprintln(os.Stderr, "    Done! Created .sadr/ in this directory.")
		_, _ = fmt.Fprintf(os.Stderr, "    Config: .sadr/config.yaml (%s preset)\n", presetName)
		_, _ = fmt.Fprintln(os.Stderr, "    Try it: run 'sadr new --quick' to capture your first snippet.")
	},
}

func addToGitignore(dir string) {
	gitignorePath := filepath.Join(dir, ".gitignore")
	entry := ".sadr/exports/"

	content, err := os.ReadFile(gitignorePath)
	if err == nil && strings.Contains(string(content), entry) {
		return
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		_, _ = f.WriteString("\n")
	}
	_, _ = f.WriteString(entry + "\n")
}

func init() {
	initCmd.Flags().StringVar(&initPreset, "preset", "minimal", "Config preset: minimal or extended")
	rootCmd.AddCommand(initCmd)
}
