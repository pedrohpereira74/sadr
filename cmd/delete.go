package cmd

import (
	"fmt"
	"os"

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
				ui.Error(os.Stderr, "provide --id <number> to delete a record.")
				return
			}

			s := storage.NewStorage(recordsDir)
			r, path, err := s.GetRecordByFileID(id)
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("record #%d not found.", id))
				return
			}

			if !opts.force {
				ui.Warning(os.Stderr, fmt.Sprintf("are you sure? this will make '%s' sadr... and gone. use --force to proceed.", r.Title))
				return
			}

			if err := s.DeleteRecord(path); err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("failed to delete: %v", err))
				return
			}

			ui.Success(os.Stderr, fmt.Sprintf("%s deleted — this will make your snippet sadr... and gone.", r.Title))
		},
	}

	cmd.Flags().StringVar(&opts.rawID, "id", "", "Record ID to delete")
	cmd.Flags().BoolVar(&opts.force, "force", false, "Skip confirmation")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Delete personal record from ~/.sadr/")
	return cmd
}

func init() {
	rootCmd.AddCommand(newDeleteCmd())
}
