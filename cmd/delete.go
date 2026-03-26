package cmd

import (
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

var deleteID int
var deleteConfirm bool
var deleteGlobal bool

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a record",
	Long:  "Permanently delete a record by its ID.",
	Run: func(cmd *cobra.Command, args []string) {
		recordsDir, err := resolveRecordsDir(deleteGlobal)
		if err != nil {
			ui.Error(os.Stderr, err.Error())
			return
		}

		s := storage.NewStorage(recordsDir)

		if deleteID > 0 {
			records, err := s.ListRecords()
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("Something went wrong: %v", err))
				return
			}
			if deleteID > len(records) {
				ui.Error(os.Stderr, fmt.Sprintf("Record #%d not found. You have %d records.", deleteID, len(records)))
				return
			}

			r := records[deleteID-1]

			if !deleteConfirm {
				ui.Warning(os.Stderr, fmt.Sprintf("Are you sure? This will make '%s' sadr... and gone. Use --confirm to proceed.", r.Title))
				return
			}

			path, err := s.GetRecordPathByIndex(deleteID - 1)
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

func init() {
	deleteCmd.Flags().IntVar(&deleteID, "id", 0, "Record ID to delete")
	deleteCmd.Flags().BoolVar(&deleteConfirm, "confirm", false, "Skip confirmation")
	deleteCmd.Flags().BoolVarP(&deleteGlobal, "global", "g", false, "Delete personal record from ~/.sadr/")
	rootCmd.AddCommand(deleteCmd)
}
