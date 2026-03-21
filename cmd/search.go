package cmd

import (
	"fmt"
	"os"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/spf13/cobra"
)

var searchDeep bool
var searchID int

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search records by title, tags, or content",
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := os.Getwd()
		paths, err := discover.FindSadrDir(dir)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		if searchID > 0 {
			s := storage.NewStorage(paths.Records)
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
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "# %s\n\n", r.Title)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", r.Type)
			if r.FileRef != "N/A" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "File: %s\n", r.FileRef)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")
			if r.Snippet != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "## Snippet\n\n```\n%s\n```\n\n", r.Snippet)
			}
			for key, value := range r.Fields {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "## %s\n\n%s\n\n", key, value)
			}
			return
		}

		if len(args) == 0 {
			_, _ = fmt.Fprintln(os.Stderr, ":(  Provide a query or use --id <number>.")
			return
		}

		query := args[0]
		results, err := search.Search(paths.Records, query, searchDeep)
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
				tags = "{no tags}"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", r.Type, r.Title, tags)
		}
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchDeep, "deep", false, "Search inside snippet content and fields")
	searchCmd.Flags().IntVar(&searchID, "id", 0, "Show a specific record by its numeric ID")
	rootCmd.AddCommand(searchCmd)
}
