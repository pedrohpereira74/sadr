package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/spf13/cobra"
)

var newTitle string
var newType string

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Capture a new record (snippet + ADR)",
	Run: func(cmd *cobra.Command, args []string) {
		if newTitle == "" {
			_, _ = fmt.Fprintln(os.Stderr, ":(  Title is required. Use --title \"your title\"")
			return
		}

		dir, _ := os.Getwd()
		paths, err := discover.FindSadrDir(dir)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		r, err := model.NewRecordWithOptions(newTitle, newType)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  %v\n", err)
			return
		}

		s := storage.NewStorage(paths.Records)
		path, err := s.SaveRecord(r)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to save: %v\n", err)
			return
		}

		_, _ = fmt.Fprintf(os.Stderr, ":(  %s saved — Congrats, your code can now defend itself in a code review.\n", filepath.Base(path))
	},
}

func init() {
	newCmd.Flags().StringVar(&newTitle, "title", "", "Record title (required)")
	newCmd.Flags().StringVar(&newType, "type", "full", "Record type: full, snippet, adr")
	rootCmd.AddCommand(newCmd)
}
