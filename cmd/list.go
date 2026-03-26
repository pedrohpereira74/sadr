package cmd

import (
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
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all records",
		Long:  "List all captured records. Can be filtered by tags or fields.",
		Run: func(cmd *cobra.Command, args []string) {
			recordsDir, err := resolveRecordsDir(opts.global)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}

			s := storage.NewStorage(recordsDir)
			records, err := s.ListRecords()
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("Something went wrong: %v", err))
				return
			}

			if opts.field != "" && len(strings.SplitN(opts.field, "=", 2)) != 2 {
				ui.Error(os.Stderr, "Invalid field filter. Use --field key=value")
				return
			}

			var filtered []struct{ Type, Title, Tags string }
			for _, r := range records {
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
					filtered = append(filtered, struct{ Type, Title, Tags string }{r.Type, r.Title, tags})
				}
			}

			if len(filtered) == 0 {
				ui.Info(os.Stderr, "Nothing here yet. Run 'sadr new' to capture your first snippet.")
				return
			}

			for _, item := range filtered {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", item.Type, item.Title, item.Tags)
			}
		},
	}

	cmd.Flags().StringVar(&opts.recordType, "type", "", "Filter by type: full, snippet, adr")
	cmd.Flags().StringVar(&opts.tags, "tags", "", "Filter by tags (comma-separated)")
	cmd.Flags().StringVar(&opts.field, "field", "", "Filter by field value (key=value)")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "List personal records from ~/.sadr/")
	return cmd
}

func init() {
	rootCmd.AddCommand(newListCmd())
}
