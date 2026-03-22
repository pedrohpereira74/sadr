package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/spf13/cobra"
)

var exportID int
var exportAll bool
var exportTags string
var exportGlobal bool

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export records to self-contained HTML",
	Run: func(cmd *cobra.Command, args []string) {
		paths, err := resolvePaths(exportGlobal)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}
		_ = os.MkdirAll(paths.Exports, 0755)

		s := storage.NewStorage(paths.Records)
		records, err := s.ListRecords()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Something went wrong: %v\n", err)
			return
		}

		if exportID > 0 {
			if exportID > len(records) {
				_, _ = fmt.Fprintf(os.Stderr, ":(  Record #%d not found. You have %d records.\n", exportID, len(records))
				return
			}
			r := records[exportID-1]
			exportRecord(r, exportID, paths.Exports)
			return
		}

		if exportTags != "" {
			count := 0
			for i, r := range records {
				if search.HasAnyTag(r.Fields["tags"], exportTags) {
					exportRecord(r, i+1, paths.Exports)
					count++
				}
			}
			if count == 0 {
				_, _ = fmt.Fprintln(os.Stderr, ":(  No records match those tags.")
			} else {
				_, _ = fmt.Fprintf(os.Stderr, ":(  %d record(s) exported.\n", count)
			}
			return
		}

		if exportAll {
			for i, r := range records {
				exportRecord(r, i+1, paths.Exports)
			}
			_, _ = fmt.Fprintf(os.Stderr, ":(  %d record(s) exported.\n", len(records))
			return
		}

		_, _ = fmt.Fprintln(os.Stderr, ":(  Provide --id <number>, --all, or --tags <list>.")
	},
}

func exportRecord(r model.Record, id int, exportsDir string) {
	var html strings.Builder
	html.WriteString("<!DOCTYPE html>\n<html><head>\n")
	html.WriteString("<meta charset=\"utf-8\">\n")
	html.WriteString(fmt.Sprintf("<title>%s</title>\n", r.Title))
	html.WriteString("<style>body{font-family:system-ui,sans-serif;max-width:800px;margin:40px auto;padding:0 20px;line-height:1.6}code{background:#f4f4f4;padding:2px 6px;border-radius:3px}pre{background:#f4f4f4;padding:16px;border-radius:6px;overflow-x:auto}.meta{color:#666;font-size:0.9em}</style>\n")
	html.WriteString("</head><body>\n")
	html.WriteString(fmt.Sprintf("<h1>%s</h1>\n", r.Title))
	html.WriteString(fmt.Sprintf("<p class=\"meta\">Type: %s", r.Type))
	if r.FileRef != "N/A" {
		html.WriteString(fmt.Sprintf(" · File: %s", r.FileRef))
	}
	html.WriteString("</p>\n")

	if r.Snippet != "" {
		html.WriteString("<h2>Snippet</h2>\n")
		html.WriteString(fmt.Sprintf("<pre><code>%s</code></pre>\n", r.Snippet))
	}

	for key, value := range r.Fields {
		html.WriteString(fmt.Sprintf("<h2>%s</h2>\n<p>%s</p>\n", key, value))
	}

	html.WriteString("</body></html>")

	slug := strings.ToLower(r.Title)
	slug = strings.ReplaceAll(slug, " ", "-")
	filename := fmt.Sprintf("sadr-%04d-%s.html", id, slug)
	outputPath := filepath.Join(exportsDir, filename)

	if err := os.WriteFile(outputPath, []byte(html.String()), 0644); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to export: %v\n", err)
		return
	}

	_, _ = fmt.Fprintf(os.Stderr, ":(  %s exported — From tribal knowledge to actual documentation.\n", filepath.Base(outputPath))
}

func init() {
	exportCmd.Flags().IntVar(&exportID, "id", 0, "Record ID to export")
	exportCmd.Flags().BoolVar(&exportAll, "all", false, "Export all records")
	exportCmd.Flags().StringVar(&exportTags, "tags", "", "Export records matching tags (comma-separated)")
	exportCmd.Flags().BoolVarP(&exportGlobal, "global", "g", false, "Export personal records from ~/.sadr/")
	rootCmd.AddCommand(exportCmd)
}
