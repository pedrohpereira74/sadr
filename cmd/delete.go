package cmd

import (
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type deleteOptions struct {
	id      int
	confirm bool
	global  bool
}

func newDeleteCmd() *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a record",
		Long:  "Permanently delete a record by its ID.",
		Run: func(cmd *cobra.Command, args []string) {
			recordsDir, err := resolveRecordsDir(opts.global)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}

			s := storage.NewStorage(recordsDir)

			if opts.id > 0 {
				records, err := s.ListRecords()
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("Something went wrong: %v", err))
					return
				}
				if opts.id > len(records) {
					ui.Error(os.Stderr, fmt.Sprintf("Record #%d not found. You have %d records.", opts.id, len(records)))
					return
				}

				r := records[opts.id-1]

				if !opts.confirm {
					ui.Warning(os.Stderr, fmt.Sprintf("Are you sure? This will make '%s' sadr... and gone. Use --confirm to proceed.", r.Title))
					return
				}

				path, err := s.GetRecordPathByIndex(opts.id - 1)
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("Something went wrong: %v", err))
					return
				}
				if err := s.DeleteRecord(path); err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("Failed to delete: %v", err))
					return
				}

				ui.Success(os.Stderr, fmt.Sprintf("%s deleted — This will make your snippet sadr... and gone.", r.Title))
				return
			}

			ui.Error(os.Stderr, "Provide --id <number> to delete a record.")
		},
	}

	cmd.Flags().IntVar(&opts.id, "id", 0, "Record ID to delete")
	cmd.Flags().BoolVar(&opts.confirm, "confirm", false, "Skip confirmation")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Delete personal record from ~/.sadr/")
	return cmd
}

func init() {
	rootCmd.AddCommand(newDeleteCmd())
}
