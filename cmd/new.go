package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/wizard"
	"github.com/spf13/cobra"
)

var newTitle string
var newQuick bool
var newGlobal bool

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Capture a new record (snippet + ADR)",
	Run:   runNew("full"),
}

var newAdrCmd = &cobra.Command{
	Use:   "adr",
	Short: "Capture an ADR only (no snippet)",
	Run:   runNew("adr"),
}

var newSnippetCmd = &cobra.Command{
	Use:   "snippet",
	Short: "Capture a snippet only (no ADR questions)",
	Run:   runNew("snippet"),
}

func runNew(recordType string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		var recordsDir string

		if newGlobal {
			home, _ := os.UserHomeDir()
			recordsDir = filepath.Join(home, ".sadr", "records")
			if _, err := os.Stat(recordsDir); os.IsNotExist(err) {
				_, _ = fmt.Fprintln(os.Stderr, ":(  Global storage not found. Run 'sadr config --global' first.")
				return
			}
		} else {
			dir, _ := os.Getwd()
			paths, err := discover.FindSadrDir(dir)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
				return
			}
			recordsDir = paths.Records
		}

		var r model.Record
		var err error

		if newTitle == "" {
			var result map[string]string
			var wizErr error

			if newQuick {
				result, wizErr = wizard.RunQuickWizard()
			} else {
				result, wizErr = wizard.RunWizard()
			}
			if wizErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, ":(  %v\n", wizErr)
				return
			}

			r, err = model.NewRecordWithOptions(result["title"], recordType)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, ":(  %v\n", err)
				return
			}
			if result["snippet"] != "" {
				r.Snippet = result["snippet"]
			}
			if result["tags"] != "" {
				r.Fields["tags"] = result["tags"]
			}
			if result["file_ref"] != "" {
				r.FileRef = result["file_ref"]
			}
		} else {
			r, err = model.NewRecordWithOptions(newTitle, recordType)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, ":(  %v\n", err)
				return
			}
		}

		s := storage.NewStorage(recordsDir)
		path, err := s.SaveRecord(r)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to save: %v\n", err)
			return
		}

		_, _ = fmt.Fprintf(os.Stderr, ":(  %s saved — Congrats, your code can now defend itself in a code review.\n", filepath.Base(path))
	}
}

func init() {
	newCmd.PersistentFlags().StringVar(&newTitle, "title", "", "Record title (skip for interactive wizard)")
	newCmd.PersistentFlags().BoolVarP(&newQuick, "quick", "q", false, "Only ask quick_fields (title + tags)")
	newCmd.PersistentFlags().BoolVarP(&newGlobal, "global", "g", false, "Save to personal vault (~/.sadr/)")
	newCmd.AddCommand(newAdrCmd)
	newCmd.AddCommand(newSnippetCmd)
	rootCmd.AddCommand(newCmd)
}
