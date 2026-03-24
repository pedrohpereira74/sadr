package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sadr",
	Short: ":( sadr — snippets + architecture decision records",
	Long:  "Capture code with context. Because snippets without a \"why\" are sadr.",
}

func init() {
	rootCmd.SetHelpTemplate(`:(  sadr — snippets + architecture decision records

Capture code with context. Because snippets without a "why" are sadr.

Usage:
  sadr <command> [flags]

Available Commands:
  init        Initialize a .sadr/ repository in the current directory
              --preset <string>    Config preset: minimal or extended (skip interactive selection)

  new         Capture a new record (snippet + ADR)
              --title <string>     Record title (skip for interactive wizard)
              -q, --quick          Only ask quick_fields (title + tags)
              -g, --global         Save to personal vault (~/.sadr/)
              -c, --clipboard      Read snippet from clipboard
              -f, --file <string>  Read snippet from file
              -s, --smart          AI suggests field values from snippet
              --diff               Read snippet from git diff

    new adr       Capture an ADR only (no snippet)
    new snippet   Capture a snippet only (no ADR questions)

  list        List all records
              --type <string>      Filter by type: full, snippet, adr
              --tags <string>      Filter by tags (comma-separated)
              --field <string>     Filter by field value (key=value)
              -g, --global         List personal records from ~/.sadr/

  search      Search records by title, tags, or content
              --deep               Search inside snippet content and fields
              --id <int>           Show a specific record by its numeric ID
              -g, --global         Search personal records in ~/.sadr/

  edit        Edit a record in $EDITOR
              --id <int>           Record ID to edit
              -g, --global         Edit personal record from ~/.sadr/

  delete      Delete a record
              --id <int>           Record ID to delete
              --confirm            Skip confirmation
              -g, --global         Delete personal record from ~/.sadr/

  export      Export records to self-contained HTML
              --id <int>           Record ID to export
              --all                Export all records
              --tags <string>      Export records matching tags (comma-separated)
              -g, --global         Export personal records from ~/.sadr/

  config      Open config in $EDITOR
              --global             Open global config (creates ~/.sadr/ on first use)
              --set-api-key <str>  Set the Gemini API key in the global config directly

Examples:
  sadr init                          Initialize sadr in current project
  sadr new --quick                   Quick capture (title + tags only)
  sadr new --smart --clipboard       AI-assisted capture from clipboard
  sadr new --smart --diff            AI-assisted capture from git diff
  sadr new snippet --file "<file>"   Capture a snippet from file
  sadr new adr                       Capture an architecture decision
  sadr list --tags "go,api"          List records filtered by tags
  sadr search "auth" --deep          Deep search across all content
  sadr search --id 3                 View record #3 in detail
  sadr edit --id 2                   Edit record #2 in your editor
  sadr export --all                  Export all records to HTML
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
