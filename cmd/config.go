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
	global              bool
	setAPIKey           string
	setupJira           bool
	setupJiraAdmin      bool
	disableJiraWarning  bool
}

func confirmOverwriteImpl() string {
	return runSelect("API key is already configured. overwrite?", []selectOption{
		{Label: "yes, overwrite", Value: "yes"},
		{Label: "no, keep existing", Value: "no"},
	})
}

func extractAPIKey(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "api_key:") && !strings.HasPrefix(trimmed, "api_key_env:") {
			val := strings.TrimPrefix(trimmed, "api_key:")
			val = strings.TrimSpace(val)
			val = strings.Trim(val, "\"")
			return val
		}
	}
	return ""
}

func newConfigCmd() *cobra.Command {
	opts := &configOptions{}

	cmd := &cobra.Command{
		Use:   "config [name]",
		Short: "Open config in $EDITOR",
		Long:  "Open a project config from .sadr/configs/. Use --global to edit your personal global configuration.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if opts.setupJiraAdmin {
				runSetupJiraAdmin()
				return
			}
			if opts.setupJira {
				runSetupJira()
				return
			}
			if opts.disableJiraWarning {
				runDisableJiraWarning()
				return
			}

			if opts.setAPIKey != "" {
				home, err := os.UserHomeDir()
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("failed to get home directory: %v", err))
					return
				}

				globalDir := filepath.Join(home, ".sadr")
				_ = os.MkdirAll(globalDir, 0700)
				configPath := filepath.Join(globalDir, "global-config.yaml")

				content, err := os.ReadFile(configPath)
				var newContent string

				if err != nil && !os.IsNotExist(err) {
					ui.Error(os.Stderr, fmt.Sprintf("failed to read config: %v", err))
					return
				}

				sanitized := strings.NewReplacer("\"", "", "\n", "", "\r", "").Replace(opts.setAPIKey)

				if err == nil {
					if existing := extractAPIKey(string(content)); existing != "" {
						if confirmOverwrite() != "yes" {
							ui.Info(os.Stderr, "cancelled. existing API key left unchanged.")
							return
						}
					}
				}

				if err != nil {
					newContent = strings.Replace(templates.GlobalConfig, `api_key: ""`+"\n", fmt.Sprintf("api_key: \"%s\"\n", sanitized), 1)
				} else {
					lines := strings.Split(string(content), "\n")
					found := false
					for i, line := range lines {
						trimmed := strings.TrimSpace(line)
						if strings.HasPrefix(trimmed, "api_key:") && !strings.HasPrefix(trimmed, "api_key_env:") {
							prefix := line[:len(line)-len(trimmed)]
							lines[i] = prefix + fmt.Sprintf("api_key: \"%s\"", sanitized)
							found = true
							break
						}
					}
					if !found {
						if len(lines) > 0 && lines[len(lines)-1] != "" {
							lines = append(lines, "")
						}
						lines = append(lines, fmt.Sprintf("ai:\n  api_key: \"%s\"", sanitized))
					}
					newContent = strings.Join(lines, "\n")
				}

				if err := os.WriteFile(configPath, []byte(newContent), 0600); err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("failed to save API key: %v", err))
					return
				}

				ui.Success(os.Stderr, "API key saved globally. the --smart mode is now fully armed and operational.")
				return
			}

			editor := findEditor()
			if editor == "" {
				ui.Error(os.Stderr, "no editor found. set $EDITOR or add 'editor' to ~/.sadr/global-config.yaml")
				return
			}

			if opts.global {
				home, err := os.UserHomeDir()
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("could not find home directory: %v", err))
					return
				}
				globalDir := filepath.Join(home, ".sadr")
				globalConfig := filepath.Join(globalDir, "global-config.yaml")

				if _, err := os.Stat(globalConfig); os.IsNotExist(err) {
					ui.Error(os.Stderr, "global config not found. run 'sadr init --global' to configure it.")
					return
				}

				if err := editorRunner(editor, globalConfig); err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("editor exited with error: %v", err))
				}
				return
			}

			dir, err := os.Getwd()
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("could not get working directory: %v", err))
				return
			}
			paths, err := discover.FindSadrDir(dir)
			if err != nil {
				ui.Error(os.Stderr, "no local sadr project found. use 'sadr config --global' to edit your personal config, or 'sadr init' to create a project here.")
				return
			}

			var targetConfig string
			if len(args) > 0 {
				targetConfig = filepath.Join(paths.ConfigsDir, configFilename(args[0]))
				if _, err := os.Stat(targetConfig); err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("config %q not found.", args[0]))
					return
				}
			} else {
				chosen, err := pickConfigFile(paths.ConfigsDir)
				if err != nil {
					if err.Error() != "cancelled" {
						ui.Error(os.Stderr, err.Error())
					}
					return
				}
				targetConfig = chosen
			}

			if err := editorRunner(editor, targetConfig); err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("editor exited with error: %v", err))
			}
		},
	}

	cmd.Flags().BoolVar(&opts.global, "global", false, "Open global config (creates ~/.sadr/ on first use)")
	cmd.Flags().StringVar(&opts.setAPIKey, "set-api-key", "", "Set the Gemini API key in the global config directly")
	cmd.Flags().BoolVar(&opts.setupJira, "setup-jira", false, "Authenticate with Jira (basic auth, bearer token, or oauth 1.0a)")
	cmd.Flags().BoolVar(&opts.setupJiraAdmin, "setup-jira-admin", false, "Generate RSA key pair for Jira OAuth 1.0a application link (run once per organization)")
	cmd.Flags().BoolVar(&opts.disableJiraWarning, "disable-jira-warning", false, "Suppress the warning shown when jira credentials exist but the project is not configured for jira")
	cmd.MarkFlagsMutuallyExclusive("global", "set-api-key", "setup-jira", "setup-jira-admin", "disable-jira-warning")
	return cmd
}

func init() {
	rootCmd.AddCommand(newConfigCmd())
}
