package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/spf13/cobra"
)

var listType string
var listTags string

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

		if listType != "" {
			var filtered []struct {
				Type, Title, Tags string
			}
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
			filterTags := strings.Split(listTags, ",")
			var filtered []struct {
				Type, Title, Tags string
			}
			for _, r := range records {
				recordTags := strings.Split(r.Fields["tags"], ",")
				if hasAnyTag(recordTags, filterTags) {
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

func hasAnyTag(recordTags []string, filterTags []string) bool {
	for _, ft := range filterTags {
		ft = strings.TrimSpace(ft)
		for _, rt := range recordTags {
			if strings.TrimSpace(rt) == ft {
				return true
			}
		}
	}
	return false
}

func init() {
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by type: full, snippet, adr")
	listCmd.Flags().StringVar(&listTags, "tags", "", "Filter by tags (comma-separated)")
	rootCmd.AddCommand(listCmd)
}
