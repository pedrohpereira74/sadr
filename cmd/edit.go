package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var editID int
var editGlobal bool

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a record in $EDITOR",
	Run: func(cmd *cobra.Command, args []string) {
		var recordsDir string

		recordsDir, err := resolveRecordsDir(editGlobal)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		if editID <= 0 {
			_, _ = fmt.Fprintln(os.Stderr, ":(  Provide --id <number> to edit a record.")
			return
		}

		entries, _ := os.ReadDir(recordsDir)
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

		path := filepath.Join(recordsDir, mdFiles[editID-1])

		editor := findEditor()
		if editor == "" {
			_, _ = fmt.Fprintln(os.Stderr, ":(  No editor found. Set $EDITOR or add 'editor' to ~/.sadr/config.yaml")
			return
		}

		openEditor(editor, path)

		_, _ = fmt.Fprintln(os.Stderr, ":(  Record updated. Git tracks the rest.")
	},
}

func init() {
	editCmd.Flags().IntVar(&editID, "id", 0, "Record ID to edit")
	editCmd.Flags().BoolVarP(&editGlobal, "global", "g", false, "Edit personal record from ~/.sadr/")
	rootCmd.AddCommand(editCmd)
}
