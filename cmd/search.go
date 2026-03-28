package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/muesli/reflow/wordwrap"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type searchOptions struct {
	deep   bool
	rawID  string
	global bool
}

func newSearchCmd() *cobra.Command {
	opts := &searchOptions{}

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search records by title, tags, or content",
		Long:  "Search your records. Use --deep to search inside snippet bodies.",
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
			if id > 0 && len(args) > 0 {
				ui.Error(os.Stderr, "cannot use --id and a search query at the same time.")
				return
			}

			if id > 0 {
				s := storage.NewStorage(recordsDir)
				r, _, err := s.GetRecordByFileID(id)
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("record #%d not found.", id))
					return
				}

				width, _, err := term.GetSize(int(os.Stdout.Fd()))
				if err != nil || width <= 0 {
					width = 80
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "# %s\n\n", r.Title)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", r.Type)
				if r.FileRef != model.NoFileRef {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "File: %s\n", r.FileRef)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")
				if r.Snippet != "" {
					wSnippet := wordwrap.String(r.Snippet, width)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "## Snippet\n\n```\n%s\n```\n\n", wSnippet)
				}

				written := map[string]bool{}
				for _, key := range r.FieldOrder {
					value, ok := r.Fields[key]
					if !ok || value == "" {
						continue
					}
					wValue := wordwrap.String(value, width)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "## %s\n\n%s\n\n", key, wValue)
					written[key] = true
				}
				var remaining []string
				for key := range r.Fields {
					if !written[key] && r.Fields[key] != "" {
						remaining = append(remaining, key)
					}
				}
				sort.Strings(remaining)
				for _, key := range remaining {
					wValue := wordwrap.String(r.Fields[key], width)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "## %s\n\n%s\n\n", key, wValue)
				}
				return
			}

			if len(args) == 0 {
				ui.Error(os.Stderr, "provide a query or use --id <number>.")
				return
			}

			query := args[0]
			s := storage.NewStorage(recordsDir)
			records, err := s.ListRecords()
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("something went wrong: %v", err))
				return
			}

			results := search.Search(records, query, opts.deep)

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
	cmd.Flags().StringVar(&opts.rawID, "id", "", "Show a specific record by its numeric ID")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Search personal records in ~/.sadr/")
	cmd.MarkFlagsMutuallyExclusive("id", "deep")
	return cmd
}

func init() {
	rootCmd.AddCommand(newSearchCmd())
}
