package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/spf13/cobra"
)

var exportID int

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export records to self-contained HTML",
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := os.Getwd()
		paths, err := discover.FindSadrDir(dir)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
			return
		}

		s := storage.NewStorage(paths.Records)

		if exportID <= 0 {
			_, _ = fmt.Fprintln(os.Stderr, ":(  Provide --id <number> to export a record.")
			return
		}

		records, err := s.ListRecords()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Something went wrong: %v\n", err)
			return
		}

		if exportID > len(records) {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Record #%d not found. You have %d records.\n", exportID, len(records))
			return
		}

		r := records[exportID-1]

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
		filename := fmt.Sprintf("sadr-%04d-%s.html", exportID, slug)
		outputPath := filepath.Join(paths.Exports, filename)

		if err := os.WriteFile(outputPath, []byte(html.String()), 0644); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to export: %v\n", err)
			return
		}

		_, _ = fmt.Fprintf(os.Stderr, ":(  %s exported — From tribal knowledge to actual documentation.\n", filepath.Base(outputPath))
	},
}

func init() {
	exportCmd.Flags().IntVar(&exportID, "id", 0, "Record ID to export")
	rootCmd.AddCommand(exportCmd)
}
