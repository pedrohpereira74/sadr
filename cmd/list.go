package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/spf13/cobra"
)

var listType string
var listTags string
var listField string
var listGlobal bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all records",
	Run: func(cmd *cobra.Command, args []string) {
		recordsDir, err := resolveRecordsDir(listGlobal)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		s := storage.NewStorage(recordsDir)
		records, err := s.ListRecords()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Something went wrong: %v\n", err)
			return
		}

		if listType != "" {
			var filtered []struct{ Type, Title, Tags string }
			for _, r := range records {
				if r.Type == listType {
					filtered = append(filtered, struct{ Type, Title, Tags string }{r.Type, r.Title, r.Fields["tags"]})
				}
			}
			if len(filtered) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, ":(  Nothing here yet. Run 'sadr new' to capture your first snippet.")
				return
			}
			for _, r := range filtered {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", r.Type, r.Title, r.Tags)
			}
			return
		}

		if listTags != "" {
			var filtered []struct{ Type, Title, Tags string }
			for _, r := range records {
				if search.HasAnyTag(r.Fields["tags"], listTags) {
					filtered = append(filtered, struct{ Type, Title, Tags string }{r.Type, r.Title, r.Fields["tags"]})
				}
			}
			if len(filtered) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, ":(  Nothing here yet. Run 'sadr new' to capture your first snippet.")
				return
			}
			for _, r := range filtered {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", r.Type, r.Title, r.Tags)
			}
			return
		}

		if listField != "" {
			parts := strings.SplitN(listField, "=", 2)
			if len(parts) != 2 {
				_, _ = fmt.Fprintln(os.Stderr, ":(  Invalid field filter. Use --field key=value")
				return
			}
			key, value := parts[0], parts[1]
			var filtered []struct{ Type, Title, Tags string }
			for _, r := range records {
				if r.Fields[key] == value {
					filtered = append(filtered, struct{ Type, Title, Tags string }{r.Type, r.Title, r.Fields["tags"]})
				}
			}
			if len(filtered) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, ":(  Nothing here yet. Run 'sadr new' to capture your first snippet.")
				return
			}
			for _, r := range filtered {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", r.Type, r.Title, r.Tags)
			}
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
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by type: full, snippet, adr")
	listCmd.Flags().StringVar(&listTags, "tags", "", "Filter by tags (comma-separated)")
	listCmd.Flags().StringVar(&listField, "field", "", "Filter by field value (key=value)")
	listCmd.Flags().BoolVarP(&listGlobal, "global", "g", false, "List personal records from ~/.sadr/")
	rootCmd.AddCommand(listCmd)
}
