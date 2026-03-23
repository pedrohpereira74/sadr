package cmd

import (
	"fmt"
	"os"

	"github.com/muesli/reflow/wordwrap"
	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var searchDeep bool
var searchID int
var searchGlobal bool

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search records by title, tags, or content",
	Run: func(cmd *cobra.Command, args []string) {
		recordsDir, err := resolveRecordsDir(searchGlobal)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		if searchID > 0 {
			s := storage.NewStorage(recordsDir)
			records, err := s.ListRecords()
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, ":(  Something went wrong: %v\n", err)
				return
			}
			if searchID > len(records) {
				_, _ = fmt.Fprintf(os.Stderr, ":(  Record #%d not found. You have %d records.\n", searchID, len(records))
				return
			}
			r := records[searchID-1]

			width, _, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil || width <= 0 {
				width = 80 // fallback
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "# %s\n\n", r.Title)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", r.Type)
			if r.FileRef != "N/A" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "File: %s\n", r.FileRef)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")
			if r.Snippet != "" {
				wSnippet := wordwrap.String(r.Snippet, width)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "## Snippet\n\n```\n%s\n```\n\n", wSnippet)
			}
			for key, value := range r.Fields {
				wValue := wordwrap.String(value, width)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "## %s\n\n%s\n\n", key, wValue)
			}
			return
		}

		if len(args) == 0 {
			_, _ = fmt.Fprintln(os.Stderr, ":(  Provide a query or use --id <number>.")
			return
		}

		query := args[0]
		results, err := search.Search(recordsDir, query, searchDeep)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Something went wrong: %v\n", err)
			return
		}

		if len(results) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStderr(), ":(  0 results. Your search is sadr than your snippets.")
			return
		}

		for _, r := range results {
			tags := r.Fields["tags"]
			if tags == "" {
				tags = "-"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", r.Type, r.Title, tags)
		}
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchDeep, "deep", false, "Search inside snippet content and fields")
	searchCmd.Flags().IntVar(&searchID, "id", 0, "Show a specific record by its numeric ID")
	searchCmd.Flags().BoolVarP(&searchGlobal, "global", "g", false, "Search personal records in ~/.sadr/")
	rootCmd.AddCommand(searchCmd)
}
