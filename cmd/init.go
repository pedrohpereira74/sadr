package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pedrohpereira74/sadr/internal/templates"
	"github.com/spf13/cobra"
)

var initPreset string

type presetModel struct {
	cursor  int
	chosen  string
	options []string
	done    bool
}

func newPresetModel() presetModel {
	return presetModel{
		options: []string{"Minimal (title + tags — quick capture)", "Extended (title + tags + ADR fields — full decisions)"},
	}
}

func (m presetModel) Init() tea.Cmd { return nil }

func (m presetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			if m.cursor == 0 {
				m.chosen = "minimal"
			} else {
				m.chosen = "extended"
			}
			m.done = true
			return m, tea.Quit
		default:
			return m, nil
		}
	}
	return m, nil
}

func (m presetModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n:(  Choose your starting config:\n\n")
	for i, opt := range m.options {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, opt))
	}
	return b.String()
}

func selectPreset() string {
	m := newPresetModel()
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "minimal"
	}
	final := finalModel.(presetModel)
	if final.chosen == "" {
		return ""
	}
	return final.chosen
}

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

		chosen := initPreset
		if chosen == "" {
			chosen = selectPreset()
			if chosen == "" {
				_, _ = fmt.Fprintln(os.Stderr, ":(  Cancelled.")
				return
			}
		}

		if err := os.MkdirAll(filepath.Join(sadrDir, "records"), 0755); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create .sadr/records/: %v\n", err)
			return
		}
		if err := os.MkdirAll(filepath.Join(sadrDir, "exports"), 0755); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create .sadr/exports/: %v\n", err)
			return
		}

		preset := templates.MinimalConfig
		if chosen == "extended" {
			preset = templates.ExtendedConfig
		}

		configPath := filepath.Join(sadrDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(preset), 0644); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Failed to create config: %v\n", err)
			return
		}

		addToGitignore(dir)

		_, _ = fmt.Fprintln(os.Stderr, ":(  sadr: therapy for snippets that lost their meaning.")
		_, _ = fmt.Fprintln(os.Stderr, "")
		_, _ = fmt.Fprintln(os.Stderr, "    Done! Created .sadr/ in this directory.")
		_, _ = fmt.Fprintf(os.Stderr, "    Config: .sadr/config.yaml (%s preset)\n", chosen)
		_, _ = fmt.Fprintln(os.Stderr, "    Try it: run 'sadr new --quick' to capture your first snippet.")
	},
}

func addToGitignore(dir string) {
	gitignorePath := filepath.Join(dir, ".gitignore")
	entry := ".sadr/exports/"

	content, err := os.ReadFile(gitignorePath)
	if err == nil && strings.Contains(string(content), entry) {
		return
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		_, _ = f.WriteString("\n")
	}
	_, _ = f.WriteString(entry + "\n")
}

func init() {
	initCmd.Flags().StringVar(&initPreset, "preset", "", "Config preset: minimal or extended (skip interactive selection)")
	rootCmd.AddCommand(initCmd)
}
