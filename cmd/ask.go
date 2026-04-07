package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedrohpereira74/sadr/internal/ask"
	"github.com/pedrohpereira74/sadr/internal/config"
	"github.com/pedrohpereira74/sadr/internal/enricher"
	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type askOptions struct {
	role     string
	question string
	tags     string
	field    string
	global   bool
	dryRun   bool
	complete bool
}

func newAskCmd() *cobra.Command {
	opts := &askOptions{}

	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Ask a direct question to a senior persona about your architecture",
		Long:  "Ask a question about your architecture decisions. A persona answers based on relevant records and source code.",
		Run: func(cmd *cobra.Command, args []string) {
			runAsk(opts)
		},
	}

	cmd.Flags().StringVar(&opts.role, "role", "", "Persona role (skip selector)")
	cmd.Flags().StringVar(&opts.question, "question", "", "Question to ask (skip input)")
	cmd.Flags().StringVar(&opts.tags, "tags", "", "Filter records by tags")
	cmd.Flags().StringVar(&opts.field, "field", "", "Filter records by field (key=value)")
	cmd.Flags().BoolVarP(&opts.global, "global", "g", false, "Use personal records from ~/.sadr/")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show token estimate without calling AI")
	cmd.Flags().BoolVar(&opts.complete, "complete", false, "Include compressed diff/snippet from each record")
	return cmd
}

func runAsk(opts *askOptions) {
	paths, err := resolvePaths(opts.global)
	if err != nil {
		ui.Error(os.Stderr, err.Error())
		return
	}

	projectRoot := filepath.Dir(paths.Root)

	var entries []storage.RecordEntry
	var entryErr error
	if paths.IsGlobal {
		s := storage.NewStorage(paths.Records)
		entries, entryErr = s.ListRecordEntries()
	} else {
		entries, entryErr = listAllRecordEntries(paths.Root)
	}
	if entryErr != nil {
		ui.Error(os.Stderr, fmt.Sprintf("something went wrong: %v", entryErr))
		return
	}

	if opts.field != "" && len(strings.SplitN(opts.field, "=", 2)) != 2 {
		ui.Error(os.Stderr, "invalid field filter. use --field key=value")
		return
	}

	var filtered []storage.RecordEntry
	for _, e := range entries {
		match := true
		if opts.tags != "" && !search.HasAnyTag(e.Record.Fields["tags"], opts.tags) {
			match = false
		}
		if match && opts.field != "" {
			parts := strings.SplitN(opts.field, "=", 2)
			if e.Record.Fields[parts[0]] != parts[1] {
				match = false
			}
		}
		if match {
			filtered = append(filtered, e)
		}
	}

	askCfg := loadAskConfig(paths.Root)
	cutoff := parseRangeCutoff(askCfg.Range)

	var askFiltered []storage.RecordEntry
	for _, e := range filtered {
		status := e.Record.Fields["status"]
		if status == "proposed" || status == "deprecated" || status == "superseded" {
			continue
		}
		if !cutoff.IsZero() && e.Record.CreatedAt.Before(cutoff) {
			continue
		}
		askFiltered = append(askFiltered, e)
	}
	filtered = askFiltered

	if askCfg.Limit > 0 && len(filtered) > askCfg.Limit {
		filtered = filtered[len(filtered)-askCfg.Limit:]
	}

	if len(filtered) == 0 {
		ui.Info(os.Stderr, "nothing here yet. run 'sadr new' to capture your first snippet.")
		return
	}

	var persona ask.Persona
	if opts.role != "" {
		persona = resolvePersonaByName(opts.role)
	} else {
		persona = selectPersona()
		if persona.Name == "" {
			return
		}
	}

	question := opts.question
	if question == "" {
		question = runTextarea("what would you like to know?", "e.g. how is authentication handled today?")
		if question == "" {
			return
		}
	}

	var enrichers []enricher.Enricher
	if je := loadJiraEnricher(); je != nil {
		enrichers = append(enrichers, je)
	}

	var contexts []enricher.RecordContext
	for _, e := range filtered {
		ctx := enricher.BuildContext(e.Record, enrichers, projectRoot)
		contexts = append(contexts, ctx)
	}

	var payloadSize int
	payloadSize += len(persona.Name) + len(persona.Instruction)
	payloadSize += len(question)
	payloadSize += 300
	for _, ctx := range contexts {
		payloadSize += len(ctx.RecordTitle)
		if opts.complete {
			payloadSize += len(enricher.ZipSnippet(ctx.RecordSnippet))
		}
		for _, v := range ctx.RecordFields {
			payloadSize += len(v)
		}
	}
	tokenEstimate := payloadSize / 4
	estimate := fmt.Sprintf("dry run: %d records, ~%dKB payload, ~%d tokens estimated.", len(filtered), payloadSize/1024, tokenEstimate)

	if opts.dryRun {
		ui.Info(os.Stderr, estimate)
		return
	}

	if !confirmPromptFn(estimate + " proceed?") {
		return
	}

	apiKey, aiModel := loadAIConfig()
	if apiKey == "" {
		ui.Error(os.Stderr, "this feature requires an AI API key. set it up: https://ai.google.dev")
		return
	}

	language := loadLanguageConfig()
	prompt := ask.BuildAskPrompt(persona, question, contexts, language, opts.complete)

	ui.Info(os.Stderr, fmt.Sprintf("asking %s...", persona.Name))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	response, err := generateTextFn(ctx, prompt, apiKey, aiModel, 60*time.Second)
	if err != nil {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr)
			ui.Info(os.Stderr, "cancelled.")
			return
		}
		ui.Error(os.Stderr, fmt.Sprintf("AI request failed: %v", err))
		return
	}

	answersDir := paths.Answers
	if err := os.MkdirAll(answersDir, 0755); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to create answers directory: %v", err))
		return
	}

	slug := PersonaSlug(persona.Name)
	nextID := storage.NextID(answersDir)
	filename := fmt.Sprintf("sadr-answer-%04d-%s.md", nextID, slug)
	outputPath := filepath.Join(answersDir, filename)

	content := fmt.Sprintf("**Persona:** %s  \n**Question:** %s\n\n---\n\n%s\n", persona.Name, question, response)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to save answer: %v", err))
		return
	}
	ui.Success(os.Stderr, fmt.Sprintf("answer saved to %s", filepath.Base(outputPath)))
}

func loadAskConfig(sadrRoot string) config.AskConfig {
	cfg, err := config.LoadFromFile(filepath.Join(sadrRoot, "config.yaml"))
	if err != nil {
		return config.AskConfig{Limit: 50, Range: "6m"}
	}
	ask := cfg.Ask
	if ask.Limit == 0 {
		ask.Limit = 50
	}
	if ask.Range == "" {
		ask.Range = "6m"
	}
	return ask
}

func parseRangeCutoff(rangeStr string) time.Time {
	if rangeStr == "" {
		return time.Time{}
	}
	var n int
	var unit string
	fmt.Sscanf(rangeStr, "%d%s", &n, &unit)
	if n <= 0 {
		return time.Time{}
	}
	now := time.Now()
	switch unit {
	case "d":
		return now.AddDate(0, 0, -n)
	case "w":
		return now.AddDate(0, 0, -n*7)
	case "m":
		return now.AddDate(0, -n, 0)
	case "y":
		return now.AddDate(-n, 0, 0)
	}
	return time.Time{}
}

func init() {
	rootCmd.AddCommand(newAskCmd())
}
