package wizard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
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
	Suggestions map[string]string
	width       int
	textarea    textarea.Model
}

func initTextarea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Digite aqui... (Enter confirma)"
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.SetWidth(80)
	ta.SetHeight(5)
	return ta
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
				prompt:    "Snippet",
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
			prompt:    "Snippet",
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
		case "text", "multitext", "list":
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
		Suggestions: make(map[string]string),
		textarea:    initTextarea(),
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.textarea.SetWidth(msg.Width - 4)
		return m, nil

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
			if current.fieldType == "text" {
				current.value = strings.TrimSpace(m.textarea.Value())
			}
			if current.required {
				return m, nil
			}
			current.value = "N/A"
			m.currentStep++
			if m.Completed() {
				m.done = true
				return m, tea.Quit
			}
			if m.steps[m.currentStep].fieldType == "text" {
				m.textarea.Reset()
				m.textarea.SetValue(m.steps[m.currentStep].value)
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

			if current.fieldType == "text" {
				current.value = strings.TrimSpace(m.textarea.Value())
				if current.value == ":skip" {
					current.value = "N/A"
				}
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
			if m.steps[m.currentStep].fieldType == "text" {
				m.textarea.Reset()
				m.textarea.SetValue(m.steps[m.currentStep].value)
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
			}

		default:
			if current.fieldType == "text" {
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}
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

	requiredHint := " (tab to skip)"
	if current.required {
		requiredHint = " (required)"
	}

	b.WriteString(fmt.Sprintf("\n:(  %s%s\n\n", current.prompt, requiredHint))

	if current.fieldType == "text" {
		b.WriteString(fmt.Sprintf("  \n%s\n\n", m.textarea.View()))
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
	if len(m.steps) > 0 && m.steps[0].fieldType == "text" {
		m.textarea.SetValue(m.steps[0].value)
	}
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

type Options struct {
	Quick       bool
	SkipEditor  bool
	Fields      []FieldDef
	Suggestions map[string]string
}

func Run(opts Options) (map[string]string, error) {
	var m Model
	if opts.Quick {
		m = NewQuickModel()
	} else if len(opts.Fields) > 0 {
		m = NewModelFromConfig(opts.Fields)
	} else {
		m = NewModel()
	}

	if opts.SkipEditor {
		m = removeEditorSteps(m)
	}

	if len(opts.Suggestions) > 0 {
		applySuggestions(&m, opts.Suggestions)
	}

	return runProgram(m)
}

func applySuggestions(m *Model, suggestions map[string]string) {
	for i, s := range m.steps {
		if val, ok := suggestions[s.name]; ok {
			if s.fieldType == "multiselect" && s.selectedMap != nil {
				parts := strings.Split(val, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					for j, opt := range s.options {
						if strings.EqualFold(opt, part) {
							m.steps[i].selectedMap[j] = true
						}
					}
				}
			} else {
				m.steps[i].value = val
			}
		}
	}
}
