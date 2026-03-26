package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/templates"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type configOptions struct {
	global    bool
	setAPIKey string
}

func newConfigCmd() *cobra.Command {
	opts := &configOptions{}

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Open config in $EDITOR",
		Long:  "Open the project's config.yaml. Use --global to edit your personal global configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			if opts.setAPIKey != "" {
				home, err := os.UserHomeDir()
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("Failed to get home directory: %v", err))
					return
				}

				globalDir := filepath.Join(home, ".sadr")
				_ = os.MkdirAll(globalDir, 0755)
				configPath := filepath.Join(globalDir, "global-config.yaml")

				content, err := os.ReadFile(configPath)
				var newContent string

				if err != nil && !os.IsNotExist(err) {
					ui.Error(os.Stderr, fmt.Sprintf("Failed to read config: %v", err))
					return
				}

				if err != nil {
					newContent = strings.Replace(templates.GlobalConfig, `api_key: ""`+"\n", fmt.Sprintf("api_key: \"%s\"\n", opts.setAPIKey), 1)
				} else {
					lines := strings.Split(string(content), "\n")
					found := false
					for i, line := range lines {
						trimmed := strings.TrimSpace(line)
						if strings.HasPrefix(trimmed, "api_key:") || strings.HasPrefix(trimmed, "api_key_env:") {
							prefix := line[:len(line)-len(trimmed)]
							lines[i] = prefix + fmt.Sprintf("api_key: \"%s\"", opts.setAPIKey)
							found = true
							break
						}
					}
					if !found {
						if len(lines) > 0 && lines[len(lines)-1] != "" {
							lines = append(lines, "")
						}
						lines = append(lines, fmt.Sprintf("ai:\n  api_key: \"%s\"", opts.setAPIKey))
					}
					newContent = strings.Join(lines, "\n")
				}

				if err := os.WriteFile(configPath, []byte(newContent), 0600); err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("Failed to save API key: %v", err))
					return
				}

				ui.Success(os.Stderr, "API Key saved globally. The --smart mode is now fully armed and operational.")
				return
			}

			editor := findEditor()
			if editor == "" {
				ui.Error(os.Stderr, "No editor found. Set $EDITOR or add 'editor' to ~/.sadr/global-config.yaml")
				return
			}

			if opts.global {
				home, err := os.UserHomeDir()
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("Could not find home directory: %v", err))
					return
				}
				globalDir := filepath.Join(home, ".sadr")
				globalConfig := filepath.Join(globalDir, "global-config.yaml")

				if _, err := os.Stat(globalConfig); os.IsNotExist(err) {
					ui.Error(os.Stderr, "Global config not found. Please run 'sadr init --global' to configure it.")
					return
				}

				if err := editorRunner(editor, globalConfig); err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("Editor exited with error: %v", err))
				}
				return
			}

			dir, err := os.Getwd()
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("Could not get working directory: %v", err))
				return
			}
			paths, err := discover.FindSadrDir(dir)
			if err != nil {
				ui.Error(os.Stderr, "No local sadr project found. Use 'sadr config --global' to edit your personal config, or 'sadr init' to create a project here.")
				return
			}

			localConfig := filepath.Join(paths.Root, "config.yaml")
			if err := editorRunner(editor, localConfig); err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("Editor exited with error: %v", err))
			}
		},
	}

	cmd.Flags().BoolVar(&opts.global, "global", false, "Open global config (creates ~/.sadr/ on first use)")
	cmd.Flags().StringVar(&opts.setAPIKey, "set-api-key", "", "Set the Gemini API key in the global config directly")
	cmd.MarkFlagsMutuallyExclusive("global", "set-api-key")
	return cmd
}

func init() {
	rootCmd.AddCommand(newConfigCmd())
}
