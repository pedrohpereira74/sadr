package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/templates"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type exportOptions struct {
	rawID   string
	all     bool
	tags    string
	global  bool
	adrOnly bool
}

func newExportCmd() *cobra.Command {
	opts := &exportOptions{}

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export records to self-contained HTML",
		Long:  "Export records into a single, styled HTML file.",
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
			if err := os.MkdirAll(paths.Exports, 0755); err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("failed to create exports directory: %v", err))
				return
			}

			s := storage.NewStorage(paths.Records)

			if id > 0 {
				r, _, err := s.GetRecordByFileID(id)
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("record #%d not found.", id))
					return
				}
				exportRecord(r, id, paths.Exports, opts.adrOnly)
				return
			}

			entries, err := s.ListRecordEntries()
			if err != nil {
				ui.Error(os.Stderr, fmt.Sprintf("something went wrong: %v", err))
				return
			}

			if opts.tags != "" {
				count := 0
				for _, e := range entries {
					if search.HasAnyTag(e.Record.Fields["tags"], opts.tags) {
						exportRecord(e.Record, e.FileID, paths.Exports, opts.adrOnly)
						count++
					}
				}
				if count == 0 {
					ui.Info(os.Stderr, "no records match those tags.")
				} else {
					ui.Success(os.Stderr, fmt.Sprintf("%d record(s) exported to %s", count, paths.Exports))
				}
				return
			}

			if opts.all {
				for _, e := range entries {
					exportRecord(e.Record, e.FileID, paths.Exports, opts.adrOnly)
				}
				ui.Success(os.Stderr, fmt.Sprintf("%d record(s) exported to %s", len(entries), paths.Exports))
				return
			}

			ui.Error(os.Stderr, "provide --id <number>, --all, or --tags <list>.")
		},
	}

	cmd.Flags().StringVar(&opts.rawID, "id", "", "ID to export")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Export all records")
	cmd.Flags().StringVar(&opts.tags, "tags", "", "Export records matching tags (comma-separated)")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Export personal records from ~/.sadr/")
	cmd.Flags().BoolVar(&opts.adrOnly, "adr", false, "Export only ADR fields (no snippet)")
	cmd.MarkFlagsMutuallyExclusive("id", "all", "tags")
	return cmd
}

func exportRecord(r model.Record, id int, exportsDir string, adrOnly bool) {
	data := templates.ExportData{
		Title:      r.Title,
		Type:       r.Type,
		FileRef:    r.FileRef,
		HasFileRef: r.FileRef != model.NoFileRef && r.FileRef != "",
		Snippet:    r.Snippet,
	}

	if adrOnly {
		data.Snippet = ""
	}

	written := map[string]bool{}
	if tags, ok := r.Fields["tags"]; ok && tags != "" {
		data.Tags = strings.Join(model.ParseTags(tags), ", ")
		written["tags"] = true
	}

	for _, key := range r.FieldOrder {
		value, ok := r.Fields[key]
		if !ok || value == "" || written[key] {
			continue
		}
		data.Fields = append(data.Fields, templates.ExportField{Key: key, Value: value})
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
		data.Fields = append(data.Fields, templates.ExportField{Key: key, Value: r.Fields[key]})
	}

	html, err := templates.RenderRecord(data)
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to render template: %v", err))
		return
	}

	slug := storage.Slugify(r.Title)
	var filename string
	if adrOnly {
		filename = fmt.Sprintf("sadr-export-%04d-adr-%s.html", id, slug)
	} else {
		filename = fmt.Sprintf("sadr-export-%04d-%s.html", id, slug)
	}
	outputPath := filepath.Join(exportsDir, filename)

	if err := os.WriteFile(outputPath, []byte(html), 0644); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to export: %v", err))
		return
	}

	ui.Success(os.Stderr, fmt.Sprintf("record exported to %s", filepath.Base(outputPath)))
}

func init() {
	rootCmd.AddCommand(newExportCmd())
}
