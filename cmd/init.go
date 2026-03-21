package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const minimalConfig = `# .sadr/config.yaml
# Customize fields to match your workflow.

fields:
  - name: title
    type: text
    required: true

  - name: tags
    type: multiselect
    required: true
    options: [architecture, api, database, security, performance, tooling, infrastructure, bugfix]

quick_fields: [title, tags]
`

const extendedConfig = `# .sadr/config.yaml
# Customize fields to match your workflow.

fields:
  - name: title
    type: text
    required: true

  - name: tags
    type: multiselect
    required: true
    options: [architecture, api, database, security, performance, tooling, infrastructure, bugfix]

  - name: status
    type: select
    required: true
    options: [proposed, accepted, deprecated, superseded]
    default: proposed

  - name: context
    type: text
    required: true

  - name: decision
    type: text
    required: true

  - name: alternatives
    type: text
    required: false

  - name: consequences
    type: text
    required: false

quick_fields: [title, tags]
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a .sadr/ repository in the current directory",
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := os.Getwd()
		sadrDir := filepath.Join(dir, ".sadr")

		if _, err := os.Stat(sadrDir); !os.IsNotExist(err) {
			_, _ = fmt.Fprintln(os.Stderr,
				":(  Nice try... sadr already lives here.\n    Whats next? 'git init' inside a git repo?")
			return
		}

		if err := os.MkdirAll(filepath.Join(sadrDir, "records"), 0755); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create .sadr/records/: %v\n", err)
			return
		}
		if err := os.MkdirAll(filepath.Join(sadrDir, "exports"), 0755); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create .sadr/exports/: %v\n", err)
			return
		}

		configPath := filepath.Join(sadrDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(minimalConfig), 0644); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create config: %v\n", err)
			return
		}

		_, _ = fmt.Fprintln(os.Stderr, ":(  sadr: therapy for snippets that lost their meaning.")
		_, _ = fmt.Fprintln(os.Stderr, "")
		_, _ = fmt.Fprintln(os.Stderr, "    Done! Created .sadr/ in this directory.")
		_, _ = fmt.Fprintln(os.Stderr, "    Config: .sadr/config.yaml (minimal preset)")
		_, _ = fmt.Fprintln(os.Stderr, "    Try it: run 'sadr new --quick' to capture your first snippet.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
