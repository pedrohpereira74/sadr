package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "sadr",
	Short:   "sadr — snippets + architecture decision records",
	Long:    "Capture code with context. Because snippets without a \"why\" are sadr.\nA tool to document decisions alongside code snippets.",
	Version: version,
}

func init() {
	rootCmd.SetHelpTemplate(`sadr — snippets + architecture decision records

Capture code with context. Because snippets without a "why" are sadr.

Usage:
  sadr <command> [flags]

Available Commands:
  init        Initialize a .sadr/ repository in the current directory
              --preset <string>    Config preset: minimal or extended (skip interactive selection)

  new         Capture a new record (snippet + ADR)
              --title <string>     Record title (skip for interactive wizard)

              -g, --global         Save to personal vault (~/.sadr/)
              -c, --clipboard      Read snippet from clipboard
              -f, --file <string>  Read snippet from file
              -s, --smart          AI suggests field values from snippet
              -d, --diff           Read snippet from git diff

    new adr       Capture an ADR only (no snippet)
    new snippet   Capture a snippet only (no ADR questions)

  search      Search records by title, tags, or content
              --deep               Search inside snippet content and fields
              --id <int>           Show a specific record by its numeric ID
              -g, --global         Search personal records in ~/.sadr/

  edit        Edit a record in $EDITOR
              --id <int>           Record ID to edit
              -g, --global         Edit personal record from ~/.sadr/

  delete      Delete a record
              --id <int>           Record ID to delete
              --force              Skip confirmation
              -g, --global         Delete personal record from ~/.sadr/

  export      Export records to self-contained HTML
              --id <int>           Record ID to export
              --all                Export all records
              --tags <string>      Export records matching tags (comma-separated)
              -g, --global         Export personal records from ~/.sadr/

  ask         Ask a direct question to a senior persona about your architecture
              --role <string>      Persona role (skip selector)
              --question <string>  Question to ask (skip input)
              --tags <string>      Filter records by tags
              --field <string>     Filter records by field (key=value)
              --dry-run            Show token estimate without calling AI
              -g, --global         Use personal records from ~/.sadr/

  config      Open config in $EDITOR
              --global             Open global config (creates ~/.sadr/ on first use)
              --set-api-key <str>  Set the Gemini API key in the global config directly

  doctor      Audit records against the diff and detect API contract drift (CI gatekeeper)
              --ci                 Non-interactive CI mode with structured output
              --base <string>      Base branch of the pull request (default "main")
              --apply <string>     Comma-separated drift IDs approved for rewrite

Examples:
  sadr init                          Initialize sadr in current project

  sadr new --smart --clipboard       AI-assisted capture from clipboard
  sadr new --smart --diff            AI-assisted capture from git diff
  sadr new snippet --file "<file>"   Capture a snippet from file
  sadr new adr                       Capture an architecture decision
  sadr search "auth" --deep          Deep search across all content
  sadr search --id 3                 View record #3 in detail
  sadr edit --id 2                   Edit record #2 in your editor
  sadr export --all                  Export all records to HTML
  sadr ask --role "dba"              Ask a question as a DBA persona
  sadr config --global               Setup global config
  sadr config --set-api-key "<key>"  Save Gemini API key for --smart mode
`)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
