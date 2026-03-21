package cmd

import (
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all records",
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := os.Getwd()
		paths, err := discover.FindSadrDir(dir)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		s := storage.NewStorage(paths.Records)
		records, err := s.ListRecords()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Something went wrong: %v\n", err)
			return
		}

		if len(records) == 0 {
			_, _ = fmt.Fprintln(os.Stderr, ":(  Nothing here yet. Run 'sadr new' to capture your first snippet.")
			return
		}

		for _, r := range records {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", r.Type, r.Title, r.Fields["tags"])
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
