package cmd

import (
	"fmt"
	"os"

	"github.com/muesli/reflow/wordwrap"
	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type searchOptions struct {
	deep   bool
	id     int
	global bool
}

func newSearchCmd() *cobra.Command {
	opts := &searchOptions{}

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search records by title, tags, or content",
		Long:  "Search your records. Use --deep to search inside snippet bodies.",
		Run: func(cmd *cobra.Command, args []string) {
			recordsDir, err := resolveRecordsDir(opts.global)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}
			if opts.id > 0 && len(args) > 0 {
				ui.Error(os.Stderr, "Cannot use --id and a search query at the same time.")
				return
			}

			if opts.id > 0 {
				s := storage.NewStorage(recordsDir)
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

				width, _, err := term.GetSize(int(os.Stdout.Fd()))
				if err != nil || width <= 0 {
					width = 80
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
				ui.Error(os.Stderr, "Provide a query or use --id <number>.")
				return
			}

			query := args[0]
			results, err := search.Search(recordsDir, query, opts.deep)
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("Something went wrong: %v", err))
				return
			}

			if len(results) == 0 {
				ui.Info(os.Stderr, "0 results. Your search is sadr than your snippets.")
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

	cmd.Flags().BoolVar(&opts.deep, "deep", false, "Search inside snippet content and fields")
	cmd.Flags().IntVar(&opts.id, "id", 0, "Show a specific record by its numeric ID")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Search personal records in ~/.sadr/")
	cmd.MarkFlagsMutuallyExclusive("id", "deep")
	return cmd
}

func init() {
	rootCmd.AddCommand(newSearchCmd())
}
