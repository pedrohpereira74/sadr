package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/hub"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/templates"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type exportOptions struct {
	rawID  string
	all    bool
	tags   string
	global bool
	mode   string
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

			if id == 0 && !opts.all && opts.tags == "" {
				if err := os.MkdirAll(paths.Exports, 0755); err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("failed to create exports directory: %v", err))
					return
				}
				recordDirs := resolveRecordDirs(opts.global, paths)
				exportsDir := paths.Exports
				if err := hub.Run(hub.Options{
					Mode:       hub.ModeExport,
					RecordDirs: recordDirs,
					ExportsDir: exportsDir,
					OnExport: func(entry storage.RecordEntry, mode hub.ExportMode) error {
						_, err := doExportRecord(entry.Record, entry.FileID, exportsDir, mode)
						return err
					},
				}); err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("hub error: %v", err))
				}
				return
			}

			s := storage.NewStorage(paths.Records)

			var exportMode hub.ExportMode
			switch opts.mode {
			case "adr":
				exportMode = hub.ExportAdr
			case "snippet":
				exportMode = hub.ExportSnippet
			default:
				exportMode = hub.ExportFull
			}

			if id > 0 {
				r, _, err := s.GetRecordByFileID(id)
				if err != nil {
					ui.Error(os.Stderr, fmt.Sprintf("record #%d not found.", id))
					return
				}
				exportRecord(r, id, paths.Exports, exportMode)
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
					if search.HasAnyTag(e.Record.Tags, opts.tags) {
						exportRecord(e.Record, e.FileID, paths.Exports, exportMode)
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
					exportRecord(e.Record, e.FileID, paths.Exports, exportMode)
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
	cmd.Flags().StringVar(&opts.mode, "mode", "full", "Export mode: full, adr, snippet")
	cmd.MarkFlagsMutuallyExclusive("id", "all", "tags")
	return cmd
}

func exportRecord(r model.Record, id int, exportsDir string, mode hub.ExportMode) {
	outputPath, err := doExportRecord(r, id, exportsDir, mode)
	if err != nil {
		ui.Error(os.Stderr, err.Error())
		return
	}
	ui.Success(os.Stderr, fmt.Sprintf("record exported to %s", filepath.Base(outputPath)))
}

func doExportRecord(r model.Record, id int, exportsDir string, mode hub.ExportMode) (string, error) {
	data := templates.ExportData{
		Title:      r.Title,
		Type:       r.Type,
		FileRef:    r.FileRef,
		HasFileRef: r.FileRef != model.NoFileRef && r.FileRef != "",
		Snippet:    r.Snippet,
	}

	if mode == hub.ExportAdr {
		data.Snippet = ""
	}

	if mode != hub.ExportSnippet {
		if len(r.Tags) > 0 {
			data.Tags = strings.Join(r.Tags, ", ")
		}
		if r.Status != "" {
			data.Status = r.Status
		}
		if r.FineTuningHint != "" {
			data.Question = r.FineTuningHint
		}

		written := map[string]bool{}

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
	}

	html, err := templates.RenderRecord(data)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %v", err)
	}

	slug := storage.Slugify(r.Title)
	var filename string
	switch mode {
	case hub.ExportAdr:
		filename = fmt.Sprintf("sadr-export-%04d-adr-%s.html", id, slug)
	case hub.ExportSnippet:
		filename = fmt.Sprintf("sadr-export-%04d-snippet-%s.html", id, slug)
	default:
		filename = fmt.Sprintf("sadr-export-%04d-%s.html", id, slug)
	}
	outputPath := filepath.Join(exportsDir, filename)

	if err := os.WriteFile(outputPath, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to export: %v", err)
	}

	return outputPath, nil
}

func init() {
	rootCmd.AddCommand(newExportCmd())
}
