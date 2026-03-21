package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

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

		_, _ = fmt.Fprintln(os.Stderr, ":(  sadr: therapy for snippets that lost their meaning.")
		_, _ = fmt.Fprintln(os.Stderr, "")
		_, _ = fmt.Fprintln(os.Stderr, "    Done! Created .sadr/ in this directory.")
		_, _ = fmt.Fprintln(os.Stderr, "    Try it: run 'sadr new --quick' to capture your first snippet.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
