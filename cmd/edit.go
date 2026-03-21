package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/spf13/cobra"
)

var editID int

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a record in $EDITOR",
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := os.Getwd()
		paths, err := discover.FindSadrDir(dir)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		if editID <= 0 {
			_, _ = fmt.Fprintln(os.Stderr, ":(  Provide --id <number> to edit a record.")
			return
		}

		entries, _ := os.ReadDir(paths.Records)
		var mdFiles []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				mdFiles = append(mdFiles, entry.Name())
			}
		}

		if editID > len(mdFiles) {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Record #%d not found. You have %d records.\n", editID, len(mdFiles))
			return
		}

		path := filepath.Join(paths.Records, mdFiles[editID-1])

		editor := findEditor()
		if editor == "" {
			_, _ = fmt.Fprintln(os.Stderr, ":(  No editor found. Set $EDITOR or add 'editor' to ~/.sadr/config.yaml")
			return
		}

		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Editor exited with error: %v\n", err)
			return
		}

		_, _ = fmt.Fprintf(os.Stderr, ":(  Record updated. Git tracks the rest.\n")
	},
}

func findEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	for _, fallback := range []string{"vim", "nano", "vi"} {
		if _, err := exec.LookPath(fallback); err == nil {
			return fallback
		}
	}
	return ""
}

func init() {
	editCmd.Flags().IntVar(&editID, "id", 0, "Record ID to edit")
	rootCmd.AddCommand(editCmd)
}
