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

var configGlobal bool
var configSetAPIKey string

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open config in $EDITOR",
	Run: func(cmd *cobra.Command, args []string) {
		if configSetAPIKey != "" {
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

			if err != nil && os.IsNotExist(err) {
				newContent = strings.Replace(templates.GlobalConfig, `api_key: ""`+"\n", fmt.Sprintf("api_key: \"%s\"\n", configSetAPIKey), 1)
			} else {
				lines := strings.Split(string(content), "\n")
				found := false
				for i, line := range lines {
					trimmed := strings.TrimSpace(line)
					if strings.HasPrefix(trimmed, "api_key:") || strings.HasPrefix(trimmed, "api_key_env:") {
						prefix := line[:len(line)-len(trimmed)]
						lines[i] = prefix + fmt.Sprintf("api_key: \"%s\"", configSetAPIKey)
						found = true
						break
					}
				}
				if !found {
					if len(lines) > 0 && lines[len(lines)-1] != "" {
						lines = append(lines, "")
					}
					lines = append(lines, fmt.Sprintf("ai:\n  api_key: \"%s\"", configSetAPIKey))
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

		if configGlobal {
			home, _ := os.UserHomeDir()
			globalDir := filepath.Join(home, ".sadr")
			globalConfig := filepath.Join(globalDir, "global-config.yaml")

			if _, err := os.Stat(globalConfig); os.IsNotExist(err) {
				ui.Error(os.Stderr, "Global config not found. Please run 'sadr init --global' to configure it.")
				return
			}

			openEditor(editor, globalConfig)
			return
		}

		dir, _ := os.Getwd()
		paths, err := discover.FindSadrDir(dir)
		if err != nil {
			ui.Error(os.Stderr, err.Error())
			return
		}

		localConfig := filepath.Join(paths.Root, "config.yaml")

		if _, err := os.Stat(localConfig); os.IsNotExist(err) {
			ui.Error(os.Stderr, "Config file missing. Run 'sadr init' to recreate.")
			return
		}

		openEditor(editor, localConfig)
	},
}

func init() {
	configCmd.Flags().BoolVar(&configGlobal, "global", false, "Open global config (creates ~/.sadr/ on first use)")
	configCmd.Flags().StringVar(&configSetAPIKey, "set-api-key", "", "Set the Gemini API key in the global config directly")
	rootCmd.AddCommand(configCmd)
}
