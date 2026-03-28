package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pedrohpereira74/sadr/internal/config"
	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/ui"
)

func parseID(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid id %q: must be a positive number", raw)
	}
	return n, nil
}

type selectOption struct {
	Label string
	Value string
}

type selectModel struct {
	prompt  string
	options []selectOption
	cursor  int
	chosen  string
	done    bool
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.chosen = m.options[m.cursor].Value
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n:(  %s\n\n", m.prompt))
	for i, opt := range m.options {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, opt.Label))
	}
	return b.String()
}

func runSelect(prompt string, options []selectOption) string {
	m := selectModel{prompt: prompt, options: options}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ""
	}
	return finalModel.(selectModel).chosen
}

func promptGlobalFallbackImpl() string {
	return runSelect("No local sadr project found. Fallback to global at ~/.sadr?", []selectOption{
		{Label: "Yes, use global", Value: "yes"},
		{Label: "No, cancel", Value: "no"},
	})
}

func handleGlobalFallback(paths discover.SadrPaths) error {
	if !paths.IsGlobal {
		return nil
	}
	chosen := fallbackPrompter()
	switch chosen {
	case "yes":
		return nil
	default:
		ui.Success(os.Stderr, "run 'sadr init' in your project to initialize a local sadr project.")
		return fmt.Errorf("cancelled")
	}
}

func findEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if home, err := os.UserHomeDir(); err == nil {
		globalConfigPath := filepath.Join(home, ".sadr", "global-config.yaml")
		if cfg, err := config.LoadGlobalFromFile(globalConfigPath); err == nil && cfg.Editor != "" {
			return cfg.Editor
		}
	}
	if runtime.GOOS == "windows" {
		for _, fallback := range []string{"notepad", "code"} {
			if _, err := exec.LookPath(fallback); err == nil {
				if fallback == "code" {
					return "code --wait"
				}
				return fallback
			}
		}
	} else {
		for _, fallback := range []string{"vim", "nano", "vi"} {
			if _, err := exec.LookPath(fallback); err == nil {
				return fallback
			}
		}
	}
	return ""
}

func openEditorImpl(editor string, path string) error {
	parts := strings.Fields(editor)
	var c *exec.Cmd
	if len(parts) > 1 {
		c = exec.Command(parts[0], append(parts[1:], path)...)
	} else {
		c = exec.Command(editor, path)
	}
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}

func resolvePaths(global bool) (discover.SadrPaths, error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return discover.SadrPaths{}, fmt.Errorf("could not find home directory: %v", err)
		}
		root := filepath.Join(home, ".sadr")
		if _, err := os.Stat(root); os.IsNotExist(err) {
			return discover.SadrPaths{}, fmt.Errorf("global storage not found. Run 'sadr config --global' first")
		}
		return discover.SadrPaths{
			Root:    root,
			Records: filepath.Join(root, "records"),
			Exports: filepath.Join(root, "exports"),
		}, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return discover.SadrPaths{}, fmt.Errorf("could not get working directory: %v", err)
	}
	paths, err := discover.FindSadrDir(dir)
	if err != nil {
		return paths, err
	}
	if err := handleGlobalFallback(paths); err != nil {
		return discover.SadrPaths{}, nil
	}
	return paths, nil
}
