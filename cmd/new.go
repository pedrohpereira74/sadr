package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/ai"
	"github.com/pedrohpereira74/sadr/internal/config"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/pedrohpereira74/sadr/internal/wizard"
	"github.com/spf13/cobra"
)

var newTitle string
var newGlobal bool
var newClipboard bool
var newFile string
var newSmart bool
var newModel string
var newDiff bool

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Capture a new record (snippet + ADR)",
	Long:  "Capture a code snippet with architectural context.\nUse subcommands 'adr' or 'snippet' for specialized captures.",
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

func loadFieldDefs(configPath string) ([]wizard.FieldDef, error) {
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("invalid config.yaml: %v", err)
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
	return fields, nil
}

func readSnippetFromSource() string {
	if newClipboard {
		content, err := readClipboard()
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("Clipboard not available: %v", err))
			return ""
		}
		if content == "" {
			ui.Info(os.Stderr, "Clipboard is empty.")
			return ""
		}
		return content
	}

	if newFile != "" {
		content, err := os.ReadFile(newFile)
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("Failed to read file: %v", err))
			return ""
		}
		return strings.TrimSpace(string(content))
	}

	if newDiff {
		cmd := exec.Command("git", "--no-pager", "diff", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("Git diff failed: %v", err))
			return ""
		}
		content := strings.TrimSpace(string(output))
		if content == "" {
			ui.Info(os.Stderr, "Git diff is empty. Stage or commit something first.")
			return ""
		}
		return content
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

		paths, err := resolvePaths(newGlobal)
		if err != nil {
			ui.Error(os.Stderr, err.Error())
			return
		}
		recordsDir = paths.Records
		configPath = filepath.Join(paths.Root, "config.yaml")

		var r model.Record

		snippetFromSource := readSnippetFromSource()
		if newDiff && snippetFromSource == "" {
			return
		}
		hasSnippet := snippetFromSource != ""

		if newSmart && !hasSnippet {
			ui.Info(os.Stderr, "Opening editor to capture snippet for AI analysis...")
			ui.Pause(1.5)
			content, editorErr := snippetCapturer()
			if editorErr != nil {
				ui.Error(os.Stderr, fmt.Sprintf("Failed to open editor: %v", editorErr))
				return
			}
			if strings.TrimSpace(content) == "" {
				ui.Info(os.Stderr, "--smart requires a snippet to analyze. Operation aborted.")
				return
			}

			snippetFromSource = content
			hasSnippet = true
		}

		if newModel != "" {
			newSmart = true
		}

		if newTitle == "" {
			var result map[string]string
			var wizErr error

			fieldDefs, cfgErr := loadFieldDefs(configPath)
			if cfgErr != nil {
				ui.Error(os.Stderr, fmt.Sprintf("\nCONFIGURATION ERROR\n%v\n\nPlease fix your config.yaml and try again.\n", cfgErr))
				os.Exit(1)
			}

			var suggestions map[string]string
			if newSmart && hasSnippet {
				ui.Info(os.Stderr, "Analyzing snippet...")

				var docLanguage string
				var aiKey string
				var modelName string
				var aiDepth bool

				home, err := os.UserHomeDir()
				if err == nil {
					globalConfigPath := filepath.Join(home, ".sadr", "global-config.yaml")
					if cfg, err := config.LoadGlobalFromFile(globalConfigPath); err == nil {
						if cfg.Language != "" {
							docLanguage = cfg.Language
						}
						aiKey = cfg.AI.APIKey
						modelName = cfg.AI.Model
						aiDepth = cfg.AI.AIDepth
					}
				}

				if newModel != "" {
					modelName = newModel
				}

				var fields []string
				for _, fd := range fieldDefs {
					if fd.Type != "select" && fd.Type != "multiselect" {
						fields = append(fields, fd.Name)
					}
				}
				if len(fields) == 0 {
					fields = []string{"title", "tags", "context", "decision"}
				}

				suggestions, err = ai.Suggest(snippetFromSource, fields, docLanguage, aiKey, modelName, aiDepth)
				if err != nil {
					ui.Error(os.Stderr, "AI API key not set or request failed. Falling back to manual mode.")
					ui.Info(os.Stderr, "Set it up: https://ai.google.dev\n\n    Starting wizard in 3 seconds...")
					ui.Pause(3.0)
					suggestions = nil
				}
			}

			opts := wizard.Options{
				SkipEditor:  hasSnippet,
				Fields:      fieldDefs,
				Suggestions: suggestions,
			}
			ui.Pause(1.5)
			result, wizErr = wizardRunner(opts)
			if wizErr != nil {
				return
			}

			r, err = model.NewRecordWithOptions(result["title"], recordType)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}

			for _, fd := range fieldDefs {
				if fd.Name != "title" && fd.Name != "snippet" {
					r.FieldOrder = append(r.FieldOrder, strings.TrimSpace(fd.Name))
				}
			}

			if hasSnippet {
				r.Snippet = snippetFromSource
			} else if result["snippet"] != "" {
				r.Snippet = result["snippet"]
			}

			for key, value := range result {
				key = strings.TrimSpace(key)
				if key == "title" || key == "snippet" {
					continue
				}
				if key == "file_ref" {
					r.FileRef = value
					continue
				}
				if value != "" {
					fieldType := "string"
					for _, fd := range fieldDefs {
						if strings.EqualFold(strings.TrimSpace(fd.Name), key) {
							fieldType = fd.Type
							break
						}
					}

					if fieldType == "list" && (strings.Contains(value, ",") || strings.Contains(value, ";")) {
						rawParts := strings.FieldsFunc(value, func(r rune) bool {
							return r == ',' || r == ';'
						})
						var bullets []string
						for _, p := range rawParts {
							p = strings.TrimSpace(p)
							if p != "" {
								bullets = append(bullets, "- "+p)
							}
						}
						value = strings.Join(bullets, "\n")
					} else if fieldType == "select" {
						value = fmt.Sprintf("[%s]", strings.TrimSpace(value))
					} else if fieldType == "multiselect" {
					}

					r.Fields[key] = value
				}
			}
		} else {
			r, err = model.NewRecordWithOptions(newTitle, recordType)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}
			if hasSnippet {
				r.Snippet = snippetFromSource
			}
		}

		s := storage.NewStorage(recordsDir)
		path, err := s.SaveRecord(r)
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("Failed to save: %v", err))
			return
		}

		ui.Success(os.Stderr, fmt.Sprintf("%s saved — Congrats, your code can now defend itself in a code review.", filepath.Base(path)))
	}
}

func captureSnippetFromEditorImpl() (string, error) {
	tmpFile, err := os.CreateTemp("", "sadr-snippet-*.txt")
	if err != nil {
		return "", err
	}
	tmpName := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpName) }()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vim"
	}

	c := exec.Command(editor, tmpName)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return "", err
	}

	content, err := os.ReadFile(tmpName)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

func init() {
	newCmd.PersistentFlags().StringVar(&newTitle, "title", "", "Record title (skip for interactive wizard)")
	newCmd.PersistentFlags().BoolVarP(&newGlobal, "global", "g", false, "Save to personal vault (~/.sadr/)")
	newCmd.PersistentFlags().BoolVarP(&newClipboard, "clipboard", "c", false, "Read snippet from clipboard")
	newCmd.PersistentFlags().StringVarP(&newFile, "file", "f", "", "Read snippet from file")
	newCmd.PersistentFlags().BoolVarP(&newSmart, "smart", "s", false, "AI suggests field values from snippet")
	newCmd.PersistentFlags().StringVar(&newModel, "model", "", "Override AI model (auto-enables --smart)")
	newCmd.PersistentFlags().BoolVarP(&newDiff, "diff", "d", false, "Read snippet from git diff")
	newCmd.MarkFlagsMutuallyExclusive("clipboard", "file", "diff")
	newCmd.AddCommand(newAdrCmd)
	newCmd.AddCommand(newSnippetCmd)
	rootCmd.AddCommand(newCmd)
}
