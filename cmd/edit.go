package cmd

import (
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/storage"
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

		s := storage.NewStorage(recordsDir)
		path, err := s.GetRecordPathByIndex(editID - 1)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Record #%d not found or invalid.\n", editID)
			return
		}

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
