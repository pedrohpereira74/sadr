package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
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

type newOptions struct {
	title     string
	global    bool
	clipboard bool
	file      string
	smart     bool
	model     string
	diff      bool
	config    string
}

func newNewCmd() *cobra.Command {
	opts := &newOptions{}

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Capture a new record (snippet + ADR)",
		Long:  "Capture a code snippet with architectural context.\nUse subcommands 'adr' or 'snippet' for specialized captures.",
		Run:   runNew("full", opts),
	}

	adrCmd := &cobra.Command{
		Use:   "adr",
		Short: "Capture an ADR only (no snippet)",
		Run:   runNew("adr", opts),
	}

	snippetCmd := &cobra.Command{
		Use:   "snippet",
		Short: "Capture a snippet only (no ADR questions)",
		Run:   runNew("snippet", opts),
	}

	cmd.PersistentFlags().StringVar(&opts.title, "title", "", "Record title (skip for interactive wizard)")
	cmd.PersistentFlags().BoolVarP(&opts.global, "global", "g", false, "Save to personal vault (~/.sadr/)")
	cmd.PersistentFlags().BoolVarP(&opts.clipboard, "clipboard", "c", false, "Read snippet from clipboard")
	cmd.PersistentFlags().StringVarP(&opts.file, "file", "f", "", "Read snippet from file")
	cmd.PersistentFlags().BoolVarP(&opts.smart, "smart", "s", false, "AI suggests field values from snippet")
	cmd.PersistentFlags().StringVar(&opts.model, "model", "", "Override AI model (auto-enables --smart)")
	cmd.PersistentFlags().BoolVarP(&opts.diff, "diff", "d", false, "Read snippet from git diff")
	cmd.PersistentFlags().StringVar(&opts.config, "config", "", "Use a specific config (e.g. db, api, default)")
	cmd.MarkFlagsMutuallyExclusive("clipboard", "file", "diff")
	cmd.AddCommand(adrCmd, snippetCmd)
	return cmd
}

func resolveConfigPath(configsDir, configFlag string) (string, error) {
	if configFlag != "" {
		path := filepath.Join(configsDir, configFilename(configFlag))
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("config %q not found in .sadr/configs/", configFlag)
		}
		return path, nil
	}

	entries, err := os.ReadDir(configsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("configs directory not found. run 'sadr init' first")
		}
		return "", fmt.Errorf("failed to read configs directory %q: %w", configsDir, err)
	}
	var configs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			configs = append(configs, e.Name())
		}
	}

	if len(configs) == 0 {
		return "", fmt.Errorf("no config files found in %q. run 'sadr init' first", configsDir)
	}

	if len(configs) == 1 {
		return filepath.Join(configsDir, configs[0]), nil
	}

	options := make([]selectOption, 0, len(configs))
	for _, f := range configs {
		name := configDisplayName(f)
		options = append(options, selectOption{Label: name, Value: f})
	}
	chosen := runSelect("which config?", options)
	if chosen == "" {
		return "", fmt.Errorf("cancelled")
	}
	return filepath.Join(configsDir, chosen), nil
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

func readSnippetFromSource(opts *newOptions) string {
	if opts.clipboard {
		content, err := clipboardReader()
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("clipboard not available: %v", err))
			return ""
		}
		if content == "" {
			ui.Info(os.Stderr, "clipboard is empty.")
			return ""
		}
		return content
	}

	if opts.file != "" {
		info, err := os.Stat(opts.file)
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to read file: %v", err))
			return ""
		}
		if info.Size() > 10<<20 {
			ui.Error(os.Stderr, "file exceeds 10 MB limit.")
			return ""
		}
		content, err := os.ReadFile(opts.file)
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to read file: %v", err))
			return ""
		}
		return strings.TrimSpace(string(content))
	}

	if opts.diff {
		cmd := exec.Command("git", "--no-pager", "diff", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("git diff failed: %v", err))
			return ""
		}
		content := strings.TrimSpace(string(output))
		if content == "" {
			ui.Info(os.Stderr, "git diff is empty. stage or commit something first.")
			return ""
		}
		return content
	}

	return ""
}

func readClipboardImpl() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		if _, err := exec.LookPath("wl-paste"); err == nil {
			cmd = exec.Command("wl-paste", "--no-newline")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--output")
		} else {
			return "", fmt.Errorf("install wl-clipboard (Wayland) or xclip/xsel (X11) to use --clipboard")
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

func loadAISuggestions(opts *newOptions, snippet string, fieldDefs []wizard.FieldDef) (map[string]string, error) {
	ui.Info(os.Stderr, "analyzing snippet...")

	var docLanguage, aiKey, modelName string
	var aiDepth bool

	home, err := os.UserHomeDir()
	if err == nil {
		globalConfigPath := filepath.Join(home, ".sadr", "global-config.yaml")
		if cfg, err := config.LoadGlobalFromFile(globalConfigPath); err == nil {
			docLanguage = cfg.Language
			aiKey = cfg.AI.APIKey
			if aiKey == "" && cfg.AI.APIKeyEnv != "" {
				aiKey = os.Getenv(cfg.AI.APIKeyEnv)
			}
			modelName = cfg.AI.Model
			aiDepth = cfg.AI.AIDepth
		}
	}

	if opts.model != "" {
		modelName = opts.model
	}

	var fields []string
	for _, fd := range fieldDefs {
		if fd.Type != "select" && fd.Type != "multiselect" {
			hint := fd.Name
			if fd.Type == "text" || fd.Type == "list" {
				hint = fmt.Sprintf("%s (%s)", fd.Name, fd.Type)
			}
			fields = append(fields, hint)
		}
	}
	if len(fields) == 0 {
		fields = []string{"title", "tags"}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	suggestions, err := ai.Suggest(ctx, snippet, fields, docLanguage, aiKey, modelName, aiDepth)
	if err != nil {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr)
			ui.Info(os.Stderr, "cancelled.")
			return nil, fmt.Errorf("cancelled")
		}
		ui.Error(os.Stderr, fmt.Sprintf("AI request failed: %v. falling back to manual mode.", err))
		ui.Info(os.Stderr, "starting wizard in 3 seconds...")
		ui.Pause(3.0)
		return nil, nil
	}
	return suggestions, nil
}

func buildRecordFromWizard(result map[string]string, fieldDefs []wizard.FieldDef, recordType string, snippet string) (model.Record, error) {
	r, err := model.NewRecordWithOptions(result["title"], recordType)
	if err != nil {
		return r, err
	}

	for _, fd := range fieldDefs {
		if fd.Name != "title" && fd.Name != "snippet" {
			r.FieldOrder = append(r.FieldOrder, strings.TrimSpace(fd.Name))
		}
	}

	if snippet != "" {
		r.Snippet = snippet
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
		if value == "" {
			continue
		}

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
		}

		r.Fields[key] = value
	}

	r.Fields["status"] = "active"

	return r, nil
}

func extractFilesFromDiff(diffContent string) []string {
	seen := map[string]bool{}
	var files []string
	for line := range strings.SplitSeq(diffContent, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				bPath := parts[3]
				if strings.HasPrefix(bPath, "b/") {
					path := bPath[2:]
					if !seen[path] {
						seen[path] = true
						files = append(files, path)
					}
				}
			}
		}
	}
	return files
}

func listUntrackedFiles() []string {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

func collectChangedFiles(diffContent string) []string {
	seen := map[string]bool{}
	var files []string

	for _, f := range extractFilesFromDiff(diffContent) {
		if !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}
	for _, f := range listUntrackedFiles() {
		if !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}
	return files
}

func runNew(recordType string, opts *newOptions) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		paths, err := resolvePaths(opts.global)
		if err != nil {
			ui.Error(os.Stderr, err.Error())
			return
		}
		if err := os.MkdirAll(paths.Records, 0755); err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to create records directory: %v", err))
			return
		}
		recordsDir := paths.Records
		configPath, err := resolveConfigPath(paths.ConfigsDir, opts.config)
		if err != nil {
			if err.Error() == "cancelled" {
				ui.Info(os.Stderr, "cancelled.")
				return
			}
			ui.Error(os.Stderr, err.Error())
			return
		}

		snippetFromSource := readSnippetFromSource(opts)
		if opts.diff && snippetFromSource == "" {
			return
		}
		hasSnippet := snippetFromSource != ""

		if opts.smart && !hasSnippet {
			ui.Info(os.Stderr, "opening editor to capture snippet for AI analysis...")
			ui.Pause(1.5)
			content, editorErr := snippetCapturer()
			if editorErr != nil {
				ui.Error(os.Stderr, fmt.Sprintf("failed to open editor: %v", editorErr))
				return
			}
			if strings.TrimSpace(content) == "" {
				ui.Info(os.Stderr, "--smart requires a snippet to analyze. operation aborted.")
				return
			}
			snippetFromSource = content
			hasSnippet = true
		}

		if opts.model != "" {
			opts.smart = true
		}

		projectRoot := filepath.Dir(paths.Root)

		autoFileRef := ""
		var preSelectedFiles []string
		if opts.file != "" {
			rel, relErr := filepath.Rel(projectRoot, opts.file)
			if relErr == nil {
				autoFileRef = rel
			} else {
				autoFileRef = opts.file
			}
		}
		if opts.diff && snippetFromSource != "" {
			preSelectedFiles = collectChangedFiles(snippetFromSource)
		}

		var r model.Record

		if opts.title == "" {
			fieldDefs, cfgErr := loadFieldDefs(configPath)
			if cfgErr != nil {
				ui.Error(os.Stderr, fmt.Sprintf("\nconfiguration error\n%v\n\nfix your config file at %q and try again.\n", cfgErr, configPath))
				return
			}

			var suggestions map[string]string
			if opts.smart && hasSnippet {
				var aiErr error
				suggestions, aiErr = loadAISuggestions(opts, snippetFromSource, fieldDefs)
				if aiErr != nil {
					return
				}
			}

			wizOpts := wizard.Options{
				SkipEditor:       hasSnippet,
				SkipFileRef:      autoFileRef != "" || opts.global,
				Fields:           fieldDefs,
				Suggestions:      suggestions,
				ProjectRoot:      projectRoot,
				PreSelectedFiles: preSelectedFiles,
			}
			ui.Pause(1.5)
			result, wizErr := wizardRunner(wizOpts)
			if wizErr != nil {
				if wizErr.Error() == "cancelled" {
					ui.Info(os.Stderr, "cancelled.")
				}
				return
			}

			if autoFileRef != "" {
				result["file_ref"] = autoFileRef
			}

			r, err = buildRecordFromWizard(result, fieldDefs, recordType, snippetFromSource)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}
		} else {
			r, err = model.NewRecordWithOptions(opts.title, recordType)
			if err != nil {
				ui.Error(os.Stderr, err.Error())
				return
			}
			if hasSnippet {
				r.Snippet = snippetFromSource
			}
			if autoFileRef != "" {
				r.FileRef = autoFileRef
			}
		}

		if paths.Username != "" {
			r.Author = paths.Username
		}

		s := storage.NewStorage(recordsDir)
		path, err := s.SaveRecord(r)
		if err != nil {
			ui.Error(os.Stderr, fmt.Sprintf("failed to save: %v", err))
			return
		}

		ui.Success(os.Stderr, fmt.Sprintf("record saved to %s", filepath.Base(path)))
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

	editor := findEditor()
	if editor == "" {
		return "", fmt.Errorf("no editor found. set $EDITOR or $VISUAL")
	}

	if err := openEditorImpl(editor, tmpName); err != nil {
		return "", err
	}

	content, err := os.ReadFile(tmpName)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

func init() {
	rootCmd.AddCommand(newNewCmd())
}
