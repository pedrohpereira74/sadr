package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/ui"
)

type fallbackModel struct {
	cursor  int
	chosen  string
	options []string
	done    bool
}

func (m fallbackModel) Init() tea.Cmd { return nil }

func (m fallbackModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				m.chosen = "yes"
			} else {
				m.chosen = "no"
			}
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m fallbackModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n:(  No local Sadr project found. Fallback to global at ~/.sadr?\n\n")
	for i, opt := range m.options {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, opt))
	}
	return b.String()
}

func promptGlobalFallbackImpl() string {
	m := fallbackModel{
		options: []string{"Yes, use global", "No, cancel"},
	}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ""
	}
	final := finalModel.(fallbackModel)
	return final.chosen
}

func handleGlobalFallback(paths discover.SadrPaths) {
	if !paths.IsGlobal {
		return
	}
	if os.Getenv("SADR_TEST") == "1" || flag.Lookup("test.v") != nil {
		return
	}
	chosen := fallbackPrompter()
	switch chosen {
	case "yes":
		return
	case "no", "":
		ui.Success(os.Stderr, "Run 'sadr init' in your project to initialize a local Sadr project.")
		os.Exit(0)
	}
}

func findEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
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

func openEditorImpl(editor string, path string) {
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

	if err := c.Run(); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("Editor exited with error: %v", err))
	}
}

func resolveRecordsDir(global bool) (string, error) {
	if global {
		home, _ := os.UserHomeDir()
		recordsDir := filepath.Join(home, ".sadr", "records")
		if _, err := os.Stat(recordsDir); os.IsNotExist(err) {
			return "", fmt.Errorf("Global storage not found. Run 'sadr config --global' first")
		}
		return recordsDir, nil
	}

	dir, _ := os.Getwd()
	paths, err := discover.FindSadrDir(dir)
	if err != nil {
		return "", err
	}
	handleGlobalFallback(paths)
	return paths.Records, nil
}

func resolvePaths(global bool) (discover.SadrPaths, error) {
	if global {
		home, _ := os.UserHomeDir()
		root := filepath.Join(home, ".sadr")
		if _, err := os.Stat(root); os.IsNotExist(err) {
			return discover.SadrPaths{}, fmt.Errorf("Global storage not found. Run 'sadr config --global' first")
		}
		return discover.SadrPaths{
			Root:    root,
			Records: filepath.Join(root, "records"),
			Exports: filepath.Join(root, "exports"),
		}, nil
	}

	dir, _ := os.Getwd()
	paths, err := discover.FindSadrDir(dir)
	if err != nil {
		return paths, err
	}
	handleGlobalFallback(paths)
	return paths, nil
}
