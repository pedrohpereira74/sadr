package wizard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type editorFinishedMsg struct {
	content string
	err     error
}

type step struct {
	name        string
	prompt      string
	value       string
	fieldType   string
	required    bool
	options     []string
	cursorPos   int
	selectedMap map[int]bool
}

type Model struct {
	currentStep int
	steps       []step
	done        bool
	quitting    bool
	tempFile    string
}

type FieldDef struct {
	Name     string
	Type     string
	Required bool
	Options  []string
	Default  string
}

func NewModel() Model {
	return Model{
		currentStep: 0,
		steps: []step{
			{
				name:      "snippet",
				prompt:    "Snippet (opens editor)",
				fieldType: "editor",
				required:  false,
			},
			{
				name:      "title",
				prompt:    "Title",
				fieldType: "text",
				required:  true,
			},
			{
				name:        "tags",
				prompt:      "Tags",
				fieldType:   "multiselect",
				required:    true,
				options:     []string{"architecture", "api", "database", "security", "performance", "tooling", "infrastructure", "bugfix"},
				selectedMap: map[int]bool{},
			},
			{
				name:      "file_ref",
				prompt:    "File reference (type :skip for N/A)",
				fieldType: "text",
				required:  true,
			},
		},
	}
}

func NewQuickModel() Model {
	return Model{
		currentStep: 0,
		steps: []step{
			{
				name:      "title",
				prompt:    "Title",
				fieldType: "text",
				required:  true,
			},
			{
				name:        "tags",
				prompt:      "Tags",
				fieldType:   "multiselect",
				required:    true,
				options:     []string{"architecture", "api", "database", "security", "performance", "tooling", "infrastructure", "bugfix"},
				selectedMap: map[int]bool{},
			},
		},
	}
}

func NewModelFromConfig(fields []FieldDef) Model {
	steps := []step{
		{
			name:      "snippet",
			prompt:    "Snippet (opens editor)",
			fieldType: "editor",
			required:  false,
		},
	}

	for _, f := range fields {
		s := step{
			name:     f.Name,
			prompt:   f.Name,
			required: f.Required,
		}

		switch f.Type {
		case "text", "multitext":
			s.fieldType = "text"
		case "select":
			s.fieldType = "select"
			s.options = f.Options
			s.cursorPos = 0
			if f.Default != "" {
				s.value = f.Default
			}
		case "multiselect":
			s.fieldType = "multiselect"
			s.options = f.Options
			s.selectedMap = map[int]bool{}
			s.cursorPos = 0
		case "filepath":
			s.fieldType = "text"
			s.prompt = f.Name + " (type :skip for N/A)"
		}

		steps = append(steps, s)
	}

	return Model{
		currentStep: 0,
		steps:       steps,
	}
}

func removeEditorSteps(m Model) Model {
	var filtered []step
	for _, s := range m.steps {
		if s.fieldType != "editor" {
			filtered = append(filtered, s)
		}
	}
	m.steps = filtered
	return m
}

func (m Model) Completed() bool {
	return m.currentStep >= len(m.steps)
}

func (m Model) Result() map[string]string {
	result := map[string]string{}
	for _, s := range m.steps {
		result[s.name] = s.value
	}
	return result
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Completed() || m.quitting {
		return m, tea.Quit
	}

	current := &m.steps[m.currentStep]

	switch msg := msg.(type) {
	case editorFinishedMsg:
		current.value = msg.content
		m.currentStep++
		if m.Completed() {
			m.done = true
			return m, tea.Quit
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyTab:
			if current.required {
				return m, nil
			}
			current.value = ""
			m.currentStep++
			if m.Completed() {
				m.done = true
				return m, tea.Quit
			}
			return m, nil

		case tea.KeyEnter:
			if current.fieldType == "editor" {
				tmpFile, err := os.CreateTemp("", "sadr-snippet-*.txt")
				if err != nil {
					m.currentStep++
					return m, nil
				}
				m.tempFile = tmpFile.Name()
				_ = tmpFile.Close()

				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = os.Getenv("VISUAL")
				}
				if editor == "" {
					editor = "vim"
				}

				c := exec.Command(editor, m.tempFile)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					content, readErr := os.ReadFile(m.tempFile)
					_ = os.Remove(m.tempFile)
					if err != nil || readErr != nil {
						return editorFinishedMsg{content: "", err: err}
					}
					return editorFinishedMsg{content: strings.TrimSpace(string(content))}
				})
			}

			if current.fieldType == "select" {
				current.value = current.options[current.cursorPos]
			}

			if current.fieldType == "multiselect" {
				var selected []string
				for i, opt := range current.options {
					if current.selectedMap[i] {
						selected = append(selected, opt)
					}
				}
				current.value = strings.Join(selected, ",")
			}

			if current.fieldType == "text" && current.value == ":skip" {
				current.value = "N/A"
			}

			if current.fieldType == "text" && current.value == "" {
				return m, nil
			}

			if current.value == "" && current.required {
				return m, nil
			}

			m.currentStep++
			if m.Completed() {
				m.done = true
				return m, tea.Quit
			}
			return m, nil

		case tea.KeyUp:
			if (current.fieldType == "multiselect" || current.fieldType == "select") && current.cursorPos > 0 {
				current.cursorPos--
			}

		case tea.KeyDown:
			if (current.fieldType == "multiselect" || current.fieldType == "select") && current.cursorPos < len(current.options)-1 {
				current.cursorPos++
			}

		case tea.KeySpace:
			if current.fieldType == "multiselect" {
				current.selectedMap[current.cursorPos] = !current.selectedMap[current.cursorPos]
			} else if current.fieldType == "text" {
				current.value += " "
			}

		case tea.KeyBackspace:
			if current.fieldType == "text" && len(current.value) > 0 {
				current.value = current.value[:len(current.value)-1]
			}

		case tea.KeyRunes:
			if current.fieldType == "text" {
				current.value += string(msg.Runes)
			}

		default:
			return m, nil
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ":(  Cancelled.\n"
	}
	if m.Completed() {
		return ":(  Done!\n"
	}

	current := m.steps[m.currentStep]
	var b strings.Builder

	requiredHint := " (Tab to skip)"
	if current.required {
		requiredHint = " (required)"
	}

	b.WriteString(fmt.Sprintf("\n:(  %s%s\n\n", current.prompt, requiredHint))

	if current.fieldType == "text" {
		b.WriteString(fmt.Sprintf("  > %s█\n", current.value))
	}

	if current.fieldType == "editor" {
		b.WriteString("  Press Enter to open editor\n")
	}

	if current.fieldType == "select" {
		b.WriteString("  ↑/↓ to navigate, Enter to confirm\n\n")
		for i, opt := range current.options {
			cursor := "  "
			if i == current.cursorPos {
				cursor = "> "
			}
			b.WriteString(fmt.Sprintf("  %s%s\n", cursor, opt))
		}
	}

	if current.fieldType == "multiselect" {
		b.WriteString("  Space to toggle, Enter to confirm\n\n")
		for i, opt := range current.options {
			cursor := "  "
			if i == current.cursorPos {
				cursor = "> "
			}
			checked := "[ ]"
			if current.selectedMap[i] {
				checked = "[x]"
			}
			b.WriteString(fmt.Sprintf("  %s%s %s\n", cursor, checked, opt))
		}
	}

	return b.String()
}

func runProgram(m Model) (map[string]string, error) {
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	final := finalModel.(Model)
	if final.quitting && !final.done {
		return nil, fmt.Errorf("cancelled")
	}

	return final.Result(), nil
}

func RunWizard(skipEditor bool) (map[string]string, error) {
	m := NewModel()
	if skipEditor {
		m = removeEditorSteps(m)
	}
	return runProgram(m)
}

func RunQuickWizard() (map[string]string, error) {
	return runProgram(NewQuickModel())
}

func RunWizardFromConfig(fields []FieldDef, skipEditor bool) (map[string]string, error) {
	m := NewModelFromConfig(fields)
	if skipEditor {
		m = removeEditorSteps(m)
	}
	return runProgram(m)
}
