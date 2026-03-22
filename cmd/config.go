package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/templates"
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
				_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to get home directory: %v\n", err)
				return
			}

			globalDir := filepath.Join(home, ".sadr")
			_ = os.MkdirAll(globalDir, 0755)
			configPath := filepath.Join(globalDir, "config.yaml")

			content, err := os.ReadFile(configPath)
			var newContent string

			if err != nil && os.IsNotExist(err) {
				newContent = fmt.Sprintf("editor: \"\"\napi_key: \"%s\"\n", configSetAPIKey)
			} else {
				lines := strings.Split(string(content), "\n")
				found := false
				for i, line := range lines {
					if strings.HasPrefix(strings.TrimSpace(line), "api_key:") {
						lines[i] = fmt.Sprintf("api_key: \"%s\"", configSetAPIKey)
						found = true
						break
					}
				}
				if !found {
					if len(lines) > 0 && lines[len(lines)-1] != "" {
						lines = append(lines, "")
					}
					lines = append(lines, fmt.Sprintf("api_key: \"%s\"", configSetAPIKey))
				}
				newContent = strings.Join(lines, "\n")
			}

			if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to save API key: %v\n", err)
				return
			}

			_, _ = fmt.Fprintln(os.Stderr, ":(  API Key saved globally. The --smart mode is now fully armed and operational.")
			return
		}

		editor := findEditor()
		if editor == "" {
			_, _ = fmt.Fprintln(os.Stderr, ":(  No editor found. Set $EDITOR or add 'editor' to ~/.sadr/config.yaml")
			return
		}

		if configGlobal {
			home, _ := os.UserHomeDir()
			globalDir := filepath.Join(home, ".sadr")
			globalConfig := filepath.Join(globalDir, "config.yaml")

			if _, err := os.Stat(globalConfig); os.IsNotExist(err) {
				if err := os.MkdirAll(globalDir, 0755); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create ~/.sadr/: %v\n", err)
					return
				}

				if err := os.WriteFile(globalConfig, []byte(templates.GlobalConfig), 0644); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create config: %v\n", err)
					return
				}

				_, _ = fmt.Fprintln(os.Stderr, ":(  Created ~/.sadr/config.yaml for the first time. Opening...")
			}

			openEditor(editor, globalConfig)
			return
		}

		dir, _ := os.Getwd()
		paths, err := discover.FindSadrDir(dir)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		localConfig := filepath.Join(paths.Root, "config.yaml")

		if _, err := os.Stat(localConfig); os.IsNotExist(err) {
			_, _ = fmt.Fprintln(os.Stderr, ":(  Config file missing. Run 'sadr init' to recreate.")
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
