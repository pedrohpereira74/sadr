package cmd

import (
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type editOptions struct {
	id     int
	global bool
}

func newEditCmd() *cobra.Command {
	opts := &editOptions{}

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a record in $EDITOR",
		Long:  "Open a specific record in your default editor.",
		Run: func(cmd *cobra.Command, args []string) {
			recordsDir, err := resolveRecordsDir(opts.global)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}

			if opts.id <= 0 {
				ui.Error(os.Stderr, "Provide --id <number> to edit a record.")
				return
			}

			s := storage.NewStorage(recordsDir)
			path, err := s.GetRecordPathByIndex(opts.id - 1)
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("Record #%d not found or invalid.", opts.id))
				return
			}

			editor := findEditor()
			if editor == "" {
				ui.Error(os.Stderr, "No editor found. Set $EDITOR or add 'editor' to ~/.sadr/config.yaml")
				return
			}

			if err := editorRunner(editor, path); err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("Editor exited with error: %v", err))
				return
			}

			ui.Success(os.Stderr, "Record updated. Git tracks the rest.")
		},
	}

	cmd.Flags().IntVar(&opts.id, "id", 0, "Record ID to edit")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Edit personal record from ~/.sadr/")
	return cmd
}

func init() {
	rootCmd.AddCommand(newEditCmd())
}
