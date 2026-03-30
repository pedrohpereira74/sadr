package wizard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pedrohpereira74/sadr/internal/filepicker"
)

var programRunner = runProgramImpl

type editorFinishedMsg struct {
	content string
	err     error
}

type step struct {
	name           string
	prompt         string
	value          string
	fieldType      string
	required       bool
	options        []string
	cursorPos      int
	selectedMap    map[int]bool
	allFiles       []string
	filtered       []string
	filterInput    []rune
	scrollOffset   int
	suggestedFiles []string
}

type Model struct {
	currentStep int
	steps       []step
	done        bool
	quitting    bool
	tempFile    string
	Suggestions map[string]string
	width       int
	height      int
	textarea    textarea.Model
	projectRoot string
}

func initTextarea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type here... (Enter to confirm)"
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
				name:        "file_ref",
				prompt:      "File reference (type :skip for N/A)",
				fieldType:   "filepicker",
				required:    true,
				selectedMap: map[int]bool{},
			},
		},
	}
}

func fieldDefToStep(f FieldDef) step {
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
	}

	return s
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

	fileRefStep := step{
		name:        "file_ref",
		prompt:      "File reference (type :skip for N/A)",
		fieldType:   "filepicker",
		required:    true,
		selectedMap: map[int]bool{},
	}

	// Separate fields into: title, tags, other required, optional
	var titleField, tagsField *FieldDef
	var otherRequired, optional []FieldDef

	for i := range fields {
		f := &fields[i]
		switch strings.ToLower(f.Name) {
		case "title":
			titleField = f
		case "tags":
			tagsField = f
		default:
			if f.Required {
				otherRequired = append(otherRequired, *f)
			} else {
				optional = append(optional, *f)
			}
		}
	}

	// Order: title, tags, file_ref, other required, optional
	if titleField != nil {
		steps = append(steps, fieldDefToStep(*titleField))
	}
	if tagsField != nil {
		steps = append(steps, fieldDefToStep(*tagsField))
	}
	steps = append(steps, fileRefStep)
	for _, f := range otherRequired {
		steps = append(steps, fieldDefToStep(f))
	}
	for _, f := range optional {
		steps = append(steps, fieldDefToStep(f))
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

func removeFileRefSteps(m Model) Model {
	var filtered []step
	for _, s := range m.steps {
		if s.fieldType != "filepicker" {
			filtered = append(filtered, s)
		}
	}
	m.steps = filtered
	return m
}

func (s *step) applyFilter() []string {
	query := string(s.filterInput)
	if query == "" && len(s.suggestedFiles) > 0 {
		return s.suggestedFiles
	}
	return filepicker.FilterFiles(s.allFiles, query)
}

func (s *step) filteredToAllIndex(filteredIdx int) int {
	if filteredIdx < 0 || filteredIdx >= len(s.filtered) {
		return -1
	}
	target := s.filtered[filteredIdx]
	for i, f := range s.allFiles {
		if f == target {
			return i
		}
	}
	return -1
}

// visibleFileCount returns how many file lines fit in the terminal.
// Fixed lines: 1 empty + 1 prompt + 1 empty + 1 filter "> " + 1 empty + 1 empty + 1 footer + 2 scroll hints = 9.
func (m Model) visibleFileCount() int {
	if m.height <= 0 {
		return 10
	}
	available := m.height - 9
	if available < 3 {
		return 3
	}
	return available
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

	if current.fieldType == "filepicker" && current.allFiles == nil && m.projectRoot != "" {
		files, err := filepicker.ListProjectFiles(m.projectRoot)
		if err == nil {
			current.allFiles = files

			if len(current.suggestedFiles) > 0 {
				// Mark suggested files as selected
				suggested := map[string]bool{}
				for _, f := range current.suggestedFiles {
					suggested[f] = true
				}
				for i, f := range current.allFiles {
					if suggested[f] {
						current.selectedMap[i] = true
					}
				}
				// Show suggested files initially
				current.filtered = current.suggestedFiles
			} else {
				current.filtered = files
			}
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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

			if current.fieldType == "filepicker" {
				if string(current.filterInput) == ":skip" {
					current.value = "N/A"
				} else {
					var selected []string
					for i, f := range current.allFiles {
						if current.selectedMap[i] {
							selected = append(selected, f)
						}
					}
					if len(selected) == 0 {
						return m, nil
					}
					current.value = strings.Join(selected, ",")
				}
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
			if current.fieldType == "filepicker" && current.cursorPos > 0 {
				current.cursorPos--
				if current.cursorPos < current.scrollOffset {
					current.scrollOffset = current.cursorPos
				}
			}
			if (current.fieldType == "multiselect" || current.fieldType == "select") && current.cursorPos > 0 {
				current.cursorPos--
			}

		case tea.KeyDown:
			if current.fieldType == "filepicker" && current.cursorPos < len(current.filtered)-1 {
				visible := m.visibleFileCount()
				current.cursorPos++
				if current.cursorPos >= current.scrollOffset+visible {
					current.scrollOffset = current.cursorPos - visible + 1
				}
			}
			if (current.fieldType == "multiselect" || current.fieldType == "select") && current.cursorPos < len(current.options)-1 {
				current.cursorPos++
			}

		case tea.KeySpace:
			if current.fieldType == "multiselect" {
				current.selectedMap[current.cursorPos] = !current.selectedMap[current.cursorPos]
			}
			if current.fieldType == "filepicker" && len(current.filtered) > 0 {
				idx := current.filteredToAllIndex(current.cursorPos)
				if idx >= 0 {
					current.selectedMap[idx] = !current.selectedMap[idx]
				}
			}

		default:
			if (msg.String() == " " || msg.String() == "space") && current.fieldType == "multiselect" {
				current.selectedMap[current.cursorPos] = !current.selectedMap[current.cursorPos]
				return m, nil
			}
			if (msg.String() == " " || msg.String() == "space") && current.fieldType == "filepicker" && len(current.filtered) > 0 {
				idx := current.filteredToAllIndex(current.cursorPos)
				if idx >= 0 {
					current.selectedMap[idx] = !current.selectedMap[idx]
				}
				return m, nil
			}

			if current.fieldType == "filepicker" {
				if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
					if len(current.filterInput) > 0 {
						current.filterInput = current.filterInput[:len(current.filterInput)-1]
						current.filtered = current.applyFilter()
						current.cursorPos = 0
						current.scrollOffset = 0
					}
					return m, nil
				}
				if msg.Type == tea.KeyRunes {
					current.filterInput = append(current.filterInput, msg.Runes...)
					current.filtered = current.applyFilter()
					current.cursorPos = 0
					current.scrollOffset = 0
					return m, nil
				}
				return m, nil
			}

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
		return ""
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

	if current.fieldType == "filepicker" {
		b.WriteString(fmt.Sprintf("  > %s\n\n", string(current.filterInput)))

		if len(current.filtered) == 0 && len(current.allFiles) == 0 {
			b.WriteString("  type to filter files · :skip to skip\n")
		} else if len(current.filtered) == 0 {
			b.WriteString("  no files match\n")
		} else {
			visible := m.visibleFileCount()
			end := current.scrollOffset + visible
			if end > len(current.filtered) {
				end = len(current.filtered)
			}

			if current.scrollOffset > 0 {
				b.WriteString(fmt.Sprintf("  ↑ %d more\n", current.scrollOffset))
			}

			for i := current.scrollOffset; i < end; i++ {
				cursor := "  "
				if i == current.cursorPos {
					cursor = "> "
				}
				idx := current.filteredToAllIndex(i)
				checked := "[ ]"
				if idx >= 0 && current.selectedMap[idx] {
					checked = "[x]"
				}
				b.WriteString(fmt.Sprintf("  %s%s %s\n", cursor, checked, current.filtered[i]))
			}

			remaining := len(current.filtered) - end
			if remaining > 0 {
				b.WriteString(fmt.Sprintf("  ↓ %d more\n", remaining))
			}

			selectedCount := 0
			for _, v := range current.selectedMap {
				if v {
					selectedCount++
				}
			}
			b.WriteString(fmt.Sprintf("\n  %d files · ↑/↓ navigate · %d selected · space toggle · enter confirm\n", len(current.filtered), selectedCount))
		}
	}

	if current.fieldType == "select" || current.fieldType == "multiselect" {
		for i, opt := range current.options {
			cursor := "  "
			if i == current.cursorPos {
				cursor = "> "
			}
			if current.fieldType == "multiselect" {
				checked := "[ ]"
				if current.selectedMap[i] {
					checked = "[x]"
				}
				b.WriteString(fmt.Sprintf("  %s%s %s\n", cursor, checked, opt))
			} else {
				b.WriteString(fmt.Sprintf("  %s%s\n", cursor, opt))
			}
		}

		if current.fieldType == "multiselect" {
			selectedCount := 0
			for _, v := range current.selectedMap {
				if v {
					selectedCount++
				}
			}
			b.WriteString(fmt.Sprintf("\n  ↑/↓ navigate · %d selected · space toggle · enter confirm\n", selectedCount))
		} else {
			b.WriteString("\n  ↑/↓ navigate · enter confirm\n")
		}
	}

	return b.String()
}

func runProgramImpl(m Model) (map[string]string, error) {
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
	SkipEditor       bool
	SkipFileRef      bool
	Fields           []FieldDef
	Suggestions      map[string]string
	ProjectRoot      string
	PreSelectedFiles []string
}

func Run(opts Options) (map[string]string, error) {
	var m Model
	if len(opts.Fields) > 0 {
		m = NewModelFromConfig(opts.Fields)
	} else {
		m = NewModel()
	}

	if opts.SkipEditor {
		m = removeEditorSteps(m)
	}

	if opts.SkipFileRef {
		m = removeFileRefSteps(m)
	}

	m.projectRoot = opts.ProjectRoot

	if len(opts.PreSelectedFiles) > 0 {
		applyPreSelectedFiles(&m, opts.PreSelectedFiles)
	}

	if len(opts.Suggestions) > 0 {
		applySuggestions(&m, opts.Suggestions)
	}

	return programRunner(m)
}

func applyPreSelectedFiles(m *Model, files []string) {
	for i, s := range m.steps {
		if s.fieldType == "filepicker" {
			m.steps[i].suggestedFiles = files
			m.steps[i].filtered = files
			if m.steps[i].selectedMap == nil {
				m.steps[i].selectedMap = map[int]bool{}
			}
			break
		}
	}
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
