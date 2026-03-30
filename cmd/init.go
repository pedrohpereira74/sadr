package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/templates"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type initOptions struct {
	preset string
	global bool
}

func selectPresetImpl() string {
	return runSelect("Choose your starting config:", []selectOption{
		{Label: "Minimal (title + tags — quick capture)", Value: "minimal"},
		{Label: "Extended (title + tags + ADR fields — full decisions)", Value: "extended"},
	})
}

func resolvePreset(opts *initOptions) string {
	if opts.preset != "" {
		return opts.preset
	}
	chosen := presetSelector()
	if chosen == "" {
		ui.Info(os.Stderr, "cancelled.")
	}
	return chosen
}

func presetConfig(chosen string) string {
	if chosen == "extended" {
		return templates.ExtendedConfig
	}
	return templates.MinimalConfig
}

func initGlobal(opts *initOptions) {
	home, err := os.UserHomeDir()
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("could not find home directory: %v", err))
		return
	}

	sadrDir := filepath.Join(home, ".sadr")
	globalConfigPath := filepath.Join(sadrDir, "global-config.yaml")
	projectConfigPath := filepath.Join(sadrDir, "config.yaml")

	_, globalExists := os.Stat(globalConfigPath)
	_, projectExists := os.Stat(projectConfigPath)
	if globalExists == nil && projectExists == nil {
		ui.Info(os.Stderr, "global workspace already initialized at ~/.sadr")
		return
	}

	chosen := resolvePreset(opts)
	if chosen == "" {
		return
	}

	if err := os.MkdirAll(filepath.Join(sadrDir, "records"), 0755); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create global records/: %v", err))
		return
	}
	if err := os.MkdirAll(filepath.Join(sadrDir, "exports"), 0755); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create global exports/: %v", err))
		return
	}

	if globalExists != nil {
		if err := os.WriteFile(globalConfigPath, []byte(templates.GlobalConfig), 0600); err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to create global config: %v", err))
			return
		}
	}

	if projectExists != nil {
		if err := os.WriteFile(projectConfigPath, []byte(presetConfig(chosen)), 0644); err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to create project config: %v", err))
			return
		}
	}

	ui.Success(os.Stderr, "done! global workspace and personal project initialized at ~/.sadr")
	ui.Info(os.Stderr, "now run 'sadr init' inside your project to start capturing.")
}

func initHeal(sadrDir string, opts *initOptions) {
	if err := os.MkdirAll(filepath.Join(sadrDir, "records"), 0755); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create records/: %v", err))
		return
	}
	if err := os.MkdirAll(filepath.Join(sadrDir, "exports"), 0755); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create exports/: %v", err))
		return
	}

	chosen := resolvePreset(opts)
	if chosen == "" {
		return
	}

	configPath := filepath.Join(sadrDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(presetConfig(chosen)), 0644); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create config: %v", err))
		return
	}

	ui.Success(os.Stderr, "healed! recreated missing config.yaml.")
}

func initFresh(cwd string, opts *initOptions) {
	sadrDir := filepath.Join(cwd, ".sadr")

	chosen := resolvePreset(opts)
	if chosen == "" {
		return
	}

	if err := os.MkdirAll(filepath.Join(sadrDir, "records"), 0755); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create .sadr/records/: %v", err))
		return
	}
	if err := os.MkdirAll(filepath.Join(sadrDir, "exports"), 0755); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create .sadr/exports/: %v", err))
		return
	}

	configPath := filepath.Join(sadrDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(presetConfig(chosen)), 0644); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create config: %v", err))
		return
	}

	addToGitignore(cwd)

	ui.Success(os.Stderr, "done! created .sadr in this directory.")
	ui.Info(os.Stderr, "try it: run 'sadr new' to capture your first record.")
}

func newInitCmd() *cobra.Command {
	opts := &initOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a .sadr/ repository in the current directory or globally",
		Long:  "Initialize a new .sadr/ directory in the current project, creating records, exports folders and a config.yaml file.",
		Run: func(cmd *cobra.Command, args []string) {
			if opts.global {
				initGlobal(opts)
				return
			}

			home, errHome := os.UserHomeDir()
			cwd, errCwd := os.Getwd()

			if errHome == nil && errCwd == nil && filepath.Clean(home) == filepath.Clean(cwd) {
				ui.Error(os.Stderr, "access denied: you cannot initialize a local sadr project in your home directory.\n    if you want to initialize your personal global workspace, run: sadr init --global")
				return
			}

			sadrDir := filepath.Join(cwd, ".sadr")

			if _, err := os.Stat(sadrDir); !os.IsNotExist(err) {
				configPath := filepath.Join(sadrDir, "config.yaml")
				if _, err := os.Stat(configPath); !os.IsNotExist(err) {
					ui.Info(os.Stderr, "nice try... sadr already lives here.")
					return
				}
				initHeal(sadrDir, opts)
				return
			}

			initFresh(cwd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.preset,
		"preset", "", "Config preset: minimal or extended (skip interactive selection)")
	cmd.Flags().BoolVarP(&opts.global,
		"global", "g", false, "Initialize a global Sadr workspace in your home directory")
	return cmd
}

func addToGitignore(dir string) {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return
	}

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
	rootCmd.AddCommand(newInitCmd())
}
