package cmd

import (
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type editOptions struct {
	rawID  string
	global bool
}

func newEditCmd() *cobra.Command {
	opts := &editOptions{}

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a record in $EDITOR",
		Long:  "Open a specific record in your default editor.",
		Run: func(cmd *cobra.Command, args []string) {
			id, err := parseID(opts.rawID)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}

			paths, err := resolvePaths(opts.global)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}
			recordsDir := paths.Records

			if id <= 0 {
				ui.Error(os.Stderr, "provide --id <number> to edit a record.")
				return
			}

			s := storage.NewStorage(recordsDir)
			_, path, err := s.GetRecordByFileID(id)
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("record #%d not found.", id))
				return
			}

			editor := findEditor()
			if editor == "" {
				ui.Error(os.Stderr, "no editor found. set $EDITOR or add 'editor' to ~/.sadr/config.yaml")
				return
			}

			if err := editorRunner(editor, path); err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("editor exited with error: %v", err))
				return
			}

			ui.Success(os.Stderr, "record updated. git tracks the rest.")
		},
	}

	cmd.Flags().StringVar(&opts.rawID, "id", "", "Record ID to edit")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Edit personal record from ~/.sadr/")
	return cmd
}

func init() {
	rootCmd.AddCommand(newEditCmd())
}
