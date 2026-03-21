package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/templates"
	"github.com/spf13/cobra"
)

var configGlobal bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open config in $EDITOR",
	Run: func(cmd *cobra.Command, args []string) {
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

func openEditor(editor string, path string) {
	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, ":(  Editor exited with error: %v\n", err)
	}
}

func init() {
	configCmd.Flags().BoolVar(&configGlobal, "global", false, "Open global config (creates ~/.sadr/ on first use)")
	rootCmd.AddCommand(configCmd)
}
