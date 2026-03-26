package cmd

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type exportOptions struct {
	id     int
	all    bool
	tags   string
	global bool
}

func newExportCmd() *cobra.Command {
	opts := &exportOptions{}

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export records to self-contained HTML",
		Long:  "Export records into a single, styled HTML file. Use flags to filter by ID or tags.",
		Run: func(cmd *cobra.Command, args []string) {
			paths, err := resolvePaths(opts.global)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}
			_ = os.MkdirAll(paths.Exports, 0755)

			s := storage.NewStorage(paths.Records)
			records, err := s.ListRecords()
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("Something went wrong: %v", err))
				return
			}

			if opts.id > 0 {
				path, err := s.GetRecordPathByIndex(opts.id - 1)
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("Record #%d not found or invalid.", opts.id))
					return
				}
				r, err := s.LoadRecord(path)
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("Failed to read record %d: %v", opts.id, err))
					return
				}
				exportRecord(r, opts.id, paths.Exports)
				return
			}

			if opts.tags != "" {
				count := 0
				for i, r := range records {
					if search.HasAnyTag(r.Fields["tags"], opts.tags) {
						exportRecord(r, i+1, paths.Exports)
						count++
					}
				}
				if count == 0 {
					ui.Info(os.Stderr, "No records match those tags.")
				} else {
					ui.Success(os.Stderr, fmt.Sprintf("%d record(s) exported.", count))
				}
				return
			}

			if opts.all {
				for i, r := range records {
					exportRecord(r, i+1, paths.Exports)
				}
				ui.Success(os.Stderr, fmt.Sprintf("%d record(s) exported.", len(records)))
				return
			}

			ui.Error(os.Stderr, "Provide --id <number>, --all, or --tags <list>.")
		},
	}

	cmd.Flags().IntVar(&opts.id, "id", 0, "Record ID to export")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Export all records")
	cmd.Flags().StringVar(&opts.tags, "tags", "", "Export records matching tags (comma-separated)")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Export personal records from ~/.sadr/")
	cmd.MarkFlagsMutuallyExclusive("id", "all", "tags")
	return cmd
}

func exportRecord(r model.Record, id int, exportsDir string) {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html><head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(r.Title)))

	b.WriteString(fmt.Sprintf("<meta name=\"sadr-type\" content=\"%s\">\n", html.EscapeString(r.Type)))
	if r.FileRef != "N/A" && r.FileRef != "" {
		b.WriteString(fmt.Sprintf(
			"<meta name=\"sadr-file-ref\" content=\"%s\">\n", html.EscapeString(r.FileRef)))
	}

	b.WriteString(`
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github.min.css">
<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
`)

	b.WriteString(`<style>
body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; line-height: 1.6; }
p { white-space: pre-wrap; }
code { font-family: monospace; background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
pre { background: #f6f8fa; padding: 16px; border-radius: 6px; overflow-x: auto; tab-size: 4; }
pre code {
    display: block;
    white-space: pre-wrap;
    word-break: normal;
    overflow-wrap: anywhere;
    background: transparent;
    padding: 0; }
@media print {
	@page { margin: 0; }
	body { margin: 1.5cm; max-width: 100%; }
	p { page-break-inside: avoid; }
	h1, h2 { page-break-after: avoid; }
}
</style>
`)

	b.WriteString("</head><body>\n")
	b.WriteString(fmt.Sprintf("<h1>%s</h1>\n", html.EscapeString(r.Title)))

	written := map[string]bool{}
	if tags, ok := r.Fields["tags"]; ok && tags != "" {
		b.WriteString(fmt.Sprintf("<p><strong>Tags:</strong> %s</p>\n", html.EscapeString(tags)))
		written["tags"] = true
	}

	if r.Snippet != "" {
		b.WriteString("<h2>Snippet</h2>\n")
		b.WriteString(fmt.Sprintf("<pre><code>%s</code></pre>\n", html.EscapeString(r.Snippet)))
	}

	for _, key := range r.FieldOrder {
		value, ok := r.Fields[key]
		if !ok || value == "" || written[key] {
			continue
		}
		b.WriteString(fmt.Sprintf("<h2>%s</h2>\n<p>%s</p>\n", html.EscapeString(key), html.EscapeString(value)))
		written[key] = true
	}
	for key, value := range r.Fields {
		if value == "" || written[key] {
			continue
		}
		b.WriteString(fmt.Sprintf("<h2>%s</h2>\n<p>%s</p>\n", html.EscapeString(key), html.EscapeString(value)))
	}

	b.WriteString("<script>hljs.highlightAll();</script>\n")
	b.WriteString("</body></html>")

	slug := storage.Slugify(r.Title)
	filename := fmt.Sprintf("sadr-%04d-%s.html", id, slug)
	outputPath := filepath.Join(exportsDir, filename)

	if err := os.WriteFile(outputPath, []byte(b.String()), 0644); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("Failed to export: %v", err))
		return
	}

	ui.Success(os.Stderr, fmt.Sprintf("%s exported — From tribal knowledge to actual documentation.", filepath.Base(outputPath)))
}

func init() {
	rootCmd.AddCommand(newExportCmd())
}
