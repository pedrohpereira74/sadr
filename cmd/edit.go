package cmd

import (
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

var editID int
var editGlobal bool

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a record in $EDITOR",
	Long:  "Open a specific record in your default editor.",
	Run: func(cmd *cobra.Command, args []string) {
		var recordsDir string

		recordsDir, err := resolveRecordsDir(editGlobal)
		if err != nil {
			ui.Error(os.Stderr, err.Error())
			return
		}

		if editID <= 0 {
			ui.Error(os.Stderr, "Provide --id <number> to edit a record.")
			return
		}

		s := storage.NewStorage(recordsDir)
		path, err := s.GetRecordPathByIndex(editID - 1)
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("Record #%d not found or invalid.", editID))
			return
		}

		editor := findEditor()
		if editor == "" {
			ui.Error(os.Stderr, "No editor found. Set $EDITOR or add 'editor' to ~/.sadr/config.yaml")
			return
		}

		editorRunner(editor, path)

		ui.Success(os.Stderr, "Record updated. Git tracks the rest.")
	},
}

func init() {
	editCmd.Flags().IntVar(&editID, "id", 0, "Record ID to edit")
	editCmd.Flags().BoolVarP(&editGlobal, "global", "g", false, "Edit personal record from ~/.sadr/")
	rootCmd.AddCommand(editCmd)
}
