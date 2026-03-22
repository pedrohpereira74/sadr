package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/spf13/cobra"
)

var deleteID int
var deleteConfirm bool
var deleteGlobal bool

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a record",
	Run: func(cmd *cobra.Command, args []string) {
		recordsDir, err := resolveRecordsDir(deleteGlobal)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		s := storage.NewStorage(recordsDir)

		if deleteID > 0 {
			records, err := s.ListRecords()
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, ":(  Something went wrong: %v\n", err)
				return
			}
			if deleteID > len(records) {
				_, _ = fmt.Fprintf(os.Stderr, ":(  Record #%d not found. You have %d records.\n", deleteID, len(records))
				return
			}

			r := records[deleteID-1]

			if !deleteConfirm {
				_, _ = fmt.Fprintf(os.Stderr, ":(  Are you sure? This will make '%s' sadr... and gone. Use --confirm to proceed.\n", r.Title)
				return
			}

			entries, _ := os.ReadDir(recordsDir)
			var mdFiles []string
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					mdFiles = append(mdFiles, entry.Name())
				}
			}

			path := filepath.Join(recordsDir, mdFiles[deleteID-1])
			if err := s.DeleteRecord(path); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to delete: %v\n", err)
				return
			}

			_, _ = fmt.Fprintf(os.Stderr, ":(  %s deleted — This will make your snippet sadr... and gone.\n", r.Title)
			return
		}

		_, _ = fmt.Fprintln(os.Stderr, ":(  Provide --id <number> to delete a record.")
	},
}

func init() {
	deleteCmd.Flags().IntVar(&deleteID, "id", 0, "Record ID to delete")
	deleteCmd.Flags().BoolVar(&deleteConfirm, "confirm", false, "Skip confirmation")
	deleteCmd.Flags().BoolVarP(&deleteGlobal, "global", "g", false, "Delete personal record from ~/.sadr/")
	rootCmd.AddCommand(deleteCmd)
}
