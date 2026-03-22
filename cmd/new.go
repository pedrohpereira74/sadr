package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/config"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/wizard"
	"github.com/spf13/cobra"
)

var newTitle string
var newQuick bool
var newGlobal bool
var newClipboard bool
var newFile string

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

func loadFieldDefs(configPath string) []wizard.FieldDef {
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return nil
	}

	var fields []wizard.FieldDef
	for _, f := range cfg.Fields {
		fd := wizard.FieldDef{
			Name:    f.Name,
			Type:    f.Type,
			Options: f.Options,
			Default: f.Default,
		}
		if f.Required != nil {
			fd.Required = *f.Required
		}
		fields = append(fields, fd)
	}
	return fields
}

func readSnippetFromSource() string {
	if newClipboard {
		content, err := readClipboard()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Clipboard not available: %v\n", err)
			return ""
		}
		if content == "" {
			_, _ = fmt.Fprintln(os.Stderr, ":(  Clipboard is empty.")
			return ""
		}
		return content
	}

	if newFile != "" {
		content, err := os.ReadFile(newFile)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to read file: %v\n", err)
			return ""
		}
		return strings.TrimSpace(string(content))
	}

	return ""
}

func readClipboard() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--output")
		} else {
			return "", fmt.Errorf("install xclip or xsel to use --clipboard")
		}
	case "windows":
		cmd = exec.Command("powershell", "-command", "Get-Clipboard")
	default:
		return "", fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func runNew(recordType string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		var recordsDir string
		var configPath string

		if newGlobal {
			paths, err := resolvePaths(true)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
				return
			}
			recordsDir = paths.Records
			configPath = filepath.Join(paths.Root, "config.yaml")
		} else {
			paths, err := resolvePaths(false)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, ":(  "+err.Error())
				return
			}
			recordsDir = paths.Records
			configPath = filepath.Join(paths.Root, "config.yaml")
		}

		var r model.Record
		var err error

		snippetFromSource := readSnippetFromSource()

		if newTitle == "" {
			var result map[string]string
			var wizErr error

			fieldDefs := loadFieldDefs(configPath)

			hasSnippet := snippetFromSource != ""

			if newQuick {
				result, wizErr = wizard.RunQuickWizard()
			} else if fieldDefs != nil {
				result, wizErr = wizard.RunWizardFromConfig(fieldDefs, hasSnippet)
			} else {
				result, wizErr = wizard.RunWizard(hasSnippet)
			}

			if wizErr != nil {
				return
			}

			r, err = model.NewRecordWithOptions(result["title"], recordType)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, ":(  %v\n", err)
				return
			}

			if snippetFromSource != "" {
				r.Snippet = snippetFromSource
			} else if result["snippet"] != "" {
				r.Snippet = result["snippet"]
			}

			for key, value := range result {
				if key == "title" || key == "snippet" {
					continue
				}
				if key == "file_ref" {
					r.FileRef = value
					continue
				}
				if value != "" {
					r.Fields[key] = value
				}
			}
		} else {
			r, err = model.NewRecordWithOptions(newTitle, recordType)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, ":(  %v\n", err)
				return
			}
			if snippetFromSource != "" {
				r.Snippet = snippetFromSource
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
	newCmd.PersistentFlags().BoolVarP(&newClipboard, "clipboard", "c", false, "Read snippet from clipboard")
	newCmd.PersistentFlags().StringVarP(&newFile, "file", "f", "", "Read snippet from file")
	newCmd.AddCommand(newAdrCmd)
	newCmd.AddCommand(newSnippetCmd)
	rootCmd.AddCommand(newCmd)
}
