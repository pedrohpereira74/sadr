package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/config"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/templates"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type initOptions struct {
	preset string
	global bool
}

func selectPresetImpl() string {
	return runSelect("choose your starting config:", []selectOption{
		{Label: "minimal (title + tags — quick capture)", Value: "minimal"},
		{Label: "extended (title + tags + ADR fields — full decisions)", Value: "extended"},
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
	configsDir := filepath.Join(sadrDir, "configs")
	defaultConfigPath := filepath.Join(configsDir, "default-config.yaml")

	if existingCfg, cfgErr := config.LoadGlobalFromFile(globalConfigPath); cfgErr == nil && existingCfg.Username != "" {
		ui.Info(os.Stderr, "global workspace already initialized at ~/.sadr")
		return
	}

	chosen := resolvePreset(opts)
	if chosen == "" {
		return
	}

	for _, dir := range []string{configsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to create %s: %v", dir, err))
			return
		}
	}

	username := resolveUsername()
	if username == "" {
		ui.Info(os.Stderr, "cancelled.")
		return
	}

	var cfg config.GlobalConfig
	if existing, cfgErr := config.LoadGlobalFromFile(globalConfigPath); cfgErr == nil {
		cfg = existing
	} else {
		if writeErr := os.WriteFile(globalConfigPath, []byte(templates.GlobalConfig), 0600); writeErr != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to create global config: %v", writeErr))
			return
		}
		cfg, _ = config.LoadGlobalFromFile(globalConfigPath)
	}

	cfg.Username = storage.Slugify(username)

	if writeErr := writeGlobalConfig(globalConfigPath, cfg); writeErr != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to write global config: %v", writeErr))
		return
	}

	if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
		if writeErr := os.WriteFile(defaultConfigPath, []byte(presetConfig(chosen)), 0644); writeErr != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to create default config: %v", writeErr))
			return
		}
	}

	for _, dir := range []string{
		filepath.Join(sadrDir, "records"),
		filepath.Join(sadrDir, "exports"),
		filepath.Join(sadrDir, "answers"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to create %s: %v", dir, err))
			return
		}
	}

	ui.Success(os.Stderr, fmt.Sprintf("done! global workspace initialized at ~/.sadr (user: %s)", cfg.Username))
	ui.Info(os.Stderr, "now run 'sadr init' inside your project to start capturing.")
}

func resolveUsername() string {
	var options []selectOption

	gitName := ""
	if out, err := exec.Command("git", "config", "--global", "user.name").Output(); err == nil {
		gitName = strings.TrimSpace(string(out))
	}

	if gitName != "" {
		options = append(options, selectOption{Label: fmt.Sprintf("use git username: %s", gitName), Value: gitName})
	}
	options = append(options, selectOption{Label: "enter custom username", Value: "custom"})

	chosen := runSelect("choose your username:", options)
	if chosen == "" {
		return ""
	}
	if chosen == "custom" {
		return runTextarea("enter your username:", "e.g. pedro")
	}
	return chosen
}

func writeGlobalConfig(path string, cfg config.GlobalConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "username:") {
			lines[i] = fmt.Sprintf("username: %q", cfg.Username)
			found = true
			break
		}
	}
	if !found {
		lines = append([]string{fmt.Sprintf("username: %q", cfg.Username)}, lines...)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}

func initHeal(sadrDir string, opts *initOptions) {
	configsDir := filepath.Join(sadrDir, "configs")
	defaultConfigPath := filepath.Join(configsDir, "default-config.yaml")

	if err := os.MkdirAll(configsDir, 0755); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create configs/: %v", err))
		return
	}

	oldConfigPath := filepath.Join(sadrDir, "config.yaml")
	if _, err := os.Stat(oldConfigPath); err == nil {
		content, readErr := os.ReadFile(oldConfigPath)
		if readErr == nil {
			if writeErr := os.WriteFile(defaultConfigPath, content, 0644); writeErr == nil {
				_ = os.Remove(oldConfigPath)
				ui.Success(os.Stderr, "healed! moved config.yaml to configs/default-config.yaml.")
				return
			}
		}
	}

	chosen := resolvePreset(opts)
	if chosen == "" {
		return
	}

	if err := os.WriteFile(defaultConfigPath, []byte(presetConfig(chosen)), 0644); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create config: %v", err))
		return
	}

	ui.Success(os.Stderr, "healed! recreated missing configs/default-config.yaml.")
}

func initFresh(cwd string, opts *initOptions) {
	sadrDir := filepath.Join(cwd, ".sadr")
	configsDir := filepath.Join(sadrDir, "configs")

	chosen := resolvePreset(opts)
	if chosen == "" {
		return
	}

	if err := os.MkdirAll(configsDir, 0755); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create .sadr/configs/: %v", err))
		return
	}

	defaultConfigPath := filepath.Join(configsDir, "default-config.yaml")
	if err := os.WriteFile(defaultConfigPath, []byte(presetConfig(chosen)), 0644); err != nil {
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
				defaultConfigPath := filepath.Join(sadrDir, "configs", "default-config.yaml")
				if _, err := os.Stat(defaultConfigPath); !os.IsNotExist(err) {
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
	entries := []string{".sadr/*/exports/", ".sadr/*/answers/"}

	content, err := os.ReadFile(gitignorePath)
	contentStr := ""
	if err == nil {
		contentStr = string(content)
	}

	var toAdd []string
	for _, entry := range entries {
		if !strings.Contains(contentStr, entry) {
			toAdd = append(toAdd, entry)
		}
	}
	if len(toAdd) == 0 {
		return
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
		_, _ = f.WriteString("\n")
	}
	for _, entry := range toAdd {
		_, _ = f.WriteString(entry + "\n")
	}
}

func init() {
	rootCmd.AddCommand(newInitCmd())
}
