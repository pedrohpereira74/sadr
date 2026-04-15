package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type deleteOptions struct {
	rawID  string
	force  bool
	global bool
}

func newDeleteCmd() *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a record",
		Long:  "Permanently delete a record by its ID.",
		Run: func(cmd *cobra.Command, args []string) {
			idUsername, id, err := parseUserID(opts.rawID)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}

			paths, err := resolvePaths(opts.global)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}

			if id <= 0 {
				ui.Error(os.Stderr, "provide --id <number> to delete a record.")
				return
			}

			var recordsDir string
			if opts.global {
				recordsDir = paths.Records
			} else if idUsername != "" {
				recordsDir = filepath.Join(paths.Root, idUsername, "records")
			} else {
				recordsDir = paths.Records
			}

			s := storage.NewStorage(recordsDir)
			r, path, err := s.GetRecordByFileID(id)
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("record #%d not found.", id))
				return
			}

			if !opts.force {
				chosen := runSelect(
					fmt.Sprintf("delete '%s'? this cannot be undone.", r.Title),
					[]selectOption{
						{Label: "yes, delete", Value: "yes"},
						{Label: "no, cancel", Value: "no"},
					},
				)
				if chosen != "yes" {
					ui.Info(os.Stderr, "cancelled.")
					return
				}
			}

			if err := s.DeleteRecord(path); err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("failed to delete: %v", err))
				return
			}

			ui.Success(os.Stderr, fmt.Sprintf("%s deleted.", r.Title))
		},
	}

	cmd.Flags().StringVar(&opts.rawID, "id", "", "Record ID to delete (supports name/id format)")
	cmd.Flags().BoolVar(&opts.force, "force", false, "Skip confirmation")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Delete personal record from ~/.sadr/")
	return cmd
}

func init() {
	rootCmd.AddCommand(newDeleteCmd())
}
