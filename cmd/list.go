package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type listOptions struct {
	recordType string
	tags       string
	field      string
	global     bool
	format     string
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all records",
		Long:  "List all captured records. Can be filtered by tags or fields.",
		Run: func(cmd *cobra.Command, args []string) {
			paths, err := resolvePaths(opts.global)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}

			s := storage.NewStorage(paths.Records)
			entries, err := s.ListRecordEntries()
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("something went wrong: %v", err))
				return
			}

			if opts.field != "" && len(strings.SplitN(opts.field, "=", 2)) != 2 {
				ui.Error(os.Stderr, "invalid field filter. use --field key=value")
				return
			}

			var filtered []struct {
				FileID     int
				Type, Title, Tags string
			}
			for _, e := range entries {
				r := e.Record
				tags := r.Fields["tags"]
				if tags == "" {
					tags = "-"
				}

				match := true
				if opts.recordType != "" && r.Type != opts.recordType {
					match = false
				}
				if match && opts.tags != "" && !search.HasAnyTag(r.Fields["tags"], opts.tags) {
					match = false
				}
				if match && opts.field != "" {
					parts := strings.SplitN(opts.field, "=", 2)
					if r.Fields[parts[0]] != parts[1] {
						match = false
					}
				}

				if match {
					filtered = append(filtered, struct {
						FileID     int
						Type, Title, Tags string
					}{e.FileID, r.Type, r.Title, tags})
				}
			}

			if len(filtered) == 0 {
				ui.Info(os.Stderr, "nothing here yet. run 'sadr new' to capture your first snippet.")
				return
			}

			if opts.format == "json" {
				type jsonEntry struct {
					ID    int    `json:"id"`
					Type  string `json:"type"`
					Title string `json:"title"`
					Tags  string `json:"tags"`
				}
				var entries []jsonEntry
				for _, item := range filtered {
					entries = append(entries, jsonEntry{item.FileID, item.Type, item.Title, item.Tags})
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				_ = enc.Encode(entries)
				return
			}

			for _, item := range filtered {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "#%d\t%s\t%s\t%s\n", item.FileID, item.Type, item.Title, item.Tags)
			}
		},
	}

	cmd.Flags().StringVar(&opts.recordType, "type", "", "Filter by type: full, snippet, adr")
	cmd.Flags().StringVar(&opts.tags, "tags", "", "Filter by tags (comma-separated)")
	cmd.Flags().StringVar(&opts.field, "field", "", "Filter by field value (key=value)")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "List personal records from ~/.sadr/")
	cmd.Flags().StringVar(&opts.format, "format", "", "Output format: json")
	return cmd
}

func init() {
	rootCmd.AddCommand(newListCmd())
}
