package wizard

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
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
	ta.Placeholder = "type here... (enter to confirm)"
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
				prompt:    "snippet",
				fieldType: "editor",
				required:  false,
			},
			{
				name:      "title",
				prompt:    "title",
				fieldType: "text",
				required:  true,
			},
			{
				name:        "tags",
				prompt:      "tags",
				fieldType:   "multiselect",
				required:    true,
				options:     []string{"architecture", "api", "database", "security", "performance", "tooling", "infrastructure", "bugfix"},
				selectedMap: map[int]bool{},
			},
			{
				name:        "file_ref",
				prompt:      "file reference (type :skip for N/A)",
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
			prompt:    "snippet",
			fieldType: "editor",
			required:  false,
		},
	}

	fileRefStep := step{
		name:        "file_ref",
		prompt:      "file reference (type :skip for N/A)",
		fieldType:   "filepicker",
		required:    true,
		selectedMap: map[int]bool{},
	}

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

func (m Model) visibleFileCount() int {
	if m.height <= 0 {
		return 10
	}
	available := m.height - 10
	if available < 3 {
		return 3
	}
	return available
}

func (m Model) visibleOptionCount() int {
	if m.height <= 0 {
		return 10
	}
	available := m.height - 7
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

// loadFilepickerFiles lazily loads project files into a filepicker step.
func loadFilepickerFiles(current *step, projectRoot string) {
	files, err := filepicker.ListProjectFiles(projectRoot)
	if err != nil {
		return
	}
	current.allFiles = files
	if len(current.suggestedFiles) > 0 {
		suggested := map[string]bool{}
		for _, f := range current.suggestedFiles {
			suggested[f] = true
		}
		for i, f := range current.allFiles {
			if suggested[f] {
				current.selectedMap[i] = true
			}
		}
		sorted := make([]string, len(current.suggestedFiles))
		copy(sorted, current.suggestedFiles)
		sort.Strings(sorted)
		current.filtered = sorted
	} else {
		current.filtered = files
	}
}

// advanceStep increments the current step and prepares the next one.
func (m Model) advanceStep() (tea.Model, tea.Cmd) {
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
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Completed() || m.quitting {
		return m, tea.Quit
	}

	current := &m.steps[m.currentStep]

	if current.fieldType == "filepicker" && current.allFiles == nil && m.projectRoot != "" {
		loadFilepickerFiles(current, m.projectRoot)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)
		m.textarea.SetHeight(max(2, msg.Height-8))
		return m, nil

	case editorFinishedMsg:
		current.value = msg.content
		return m.advanceStep()

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	current := &m.steps[m.currentStep]

	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEsc:
		if m.currentStep > 0 {
			m.currentStep--
			if m.steps[m.currentStep].fieldType == "text" {
				m.textarea.Reset()
				m.textarea.SetValue(m.steps[m.currentStep].value)
			}
		}
		return m, nil

	case tea.KeyTab:
		if current.fieldType == "text" {
			current.value = strings.TrimSpace(m.textarea.Value())
		}
		if current.required {
			return m, nil
		}
		current.value = "N/A"
		return m.advanceStep()

	case tea.KeyEnter:
		return m.handleEnterKey()

	case tea.KeyUp:
		if current.fieldType == "text" {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		if current.fieldType == "filepicker" && current.cursorPos > 0 {
			current.cursorPos--
			if current.cursorPos < current.scrollOffset {
				current.scrollOffset = current.cursorPos
			}
		}
		if (current.fieldType == "multiselect" || current.fieldType == "select") && current.cursorPos > 0 {
			current.cursorPos--
			if current.cursorPos < current.scrollOffset {
				current.scrollOffset = current.cursorPos
			}
		}

	case tea.KeyDown:
		if current.fieldType == "text" {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		if current.fieldType == "filepicker" && current.cursorPos < len(current.filtered)-1 {
			visible := m.visibleFileCount()
			current.cursorPos++
			if current.cursorPos >= current.scrollOffset+visible {
				current.scrollOffset = current.cursorPos - visible + 1
			}
		}
		if (current.fieldType == "multiselect" || current.fieldType == "select") && current.cursorPos < len(current.options)-1 {
			current.cursorPos++
			visible := m.visibleOptionCount()
			if current.cursorPos >= current.scrollOffset+visible {
				current.scrollOffset = current.cursorPos - visible + 1
			}
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

	return m, nil
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	current := &m.steps[m.currentStep]

	if current.fieldType == "editor" {
		tmpFile, err := os.CreateTemp("", "sadr-snippet-*.txt")
		if err != nil {
			return m.advanceStep()
		}
		m.tempFile = tmpFile.Name()
		_ = tmpFile.Close()

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			for _, candidate := range []string{"vim", "nano", "vi"} {
				if _, err := exec.LookPath(candidate); err == nil {
					editor = candidate
					break
				}
			}
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

	return m.advanceStep()
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

	backHint := ""
	if m.currentStep > 0 {
		backHint = " · esc back"
	}

	requiredHint := " (tab to skip)"
	if current.required {
		requiredHint = " (required)"
	}

	fmt.Fprintf(&b, "\n:(  %s%s\n\n", current.prompt, requiredHint)

	if current.fieldType == "text" {
		fmt.Fprintf(&b, "  \n%s\n\n", m.textarea.View())
		fmt.Fprintf(&b, "  enter confirm%s\n", backHint)
	}

	if current.fieldType == "editor" {
		fmt.Fprintf(&b, "  enter to open editor%s\n", backHint)
	}

	if current.fieldType == "filepicker" {
		fmt.Fprintf(&b, "  > %s\n\n", string(current.filterInput))

		if len(current.filtered) == 0 && len(current.allFiles) == 0 {
			fmt.Fprintf(&b, "  type to filter · :skip to skip%s\n", backHint)
		} else if len(current.filtered) == 0 {
			fmt.Fprintf(&b, "  no files match%s\n", backHint)
		} else {
			visible := m.visibleFileCount()
			end := min(current.scrollOffset+visible, len(current.filtered))

			if current.scrollOffset > 0 {
				fmt.Fprintf(&b, "  ↑ %d more\n", current.scrollOffset)
			} else {
				fmt.Fprintf(&b, "\n")
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
				fmt.Fprintf(&b, "  %s%s %s\n", cursor, checked, current.filtered[i])
			}

			remaining := len(current.filtered) - end
			if remaining > 0 {
				fmt.Fprintf(&b, "  ↓ %d more\n", remaining)
			} else {
				fmt.Fprintf(&b, "\n")
			}

			selectedCount := 0
			for _, v := range current.selectedMap {
				if v {
					selectedCount++
				}
			}
			fmt.Fprintf(&b, "\n  %d files · ↑/↓ navigate · %d selected · space toggle · enter confirm%s\n", len(current.filtered), selectedCount, backHint)
		}
	}

	if current.fieldType == "select" || current.fieldType == "multiselect" {
		visible := m.visibleOptionCount()
		end := min(current.scrollOffset+visible, len(current.options))

		if current.scrollOffset > 0 {
			fmt.Fprintf(&b, "  ↑ %d more\n", current.scrollOffset)
		}

		for i := current.scrollOffset; i < end; i++ {
			opt := current.options[i]
			cursor := "  "
			if i == current.cursorPos {
				cursor = "> "
			}
			if current.fieldType == "multiselect" {
				checked := "[ ]"
				if current.selectedMap[i] {
					checked = "[x]"
				}
				fmt.Fprintf(&b, "  %s%s %s\n", cursor, checked, opt)
			} else {
				fmt.Fprintf(&b, "  %s%s\n", cursor, opt)
			}
		}

		remaining := len(current.options) - end
		if remaining > 0 {
			fmt.Fprintf(&b, "  ↓ %d more\n", remaining)
		}

		if current.fieldType == "multiselect" {
			selectedCount := 0
			for _, v := range current.selectedMap {
				if v {
					selectedCount++
				}
			}
			fmt.Fprintf(&b, "\n  ↑/↓ navigate · %d selected · space toggle · enter confirm%s\n", selectedCount, backHint)
		} else {
			fmt.Fprintf(&b, "\n  ↑/↓ navigate · enter confirm%s\n", backHint)
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
		applyPreSelectedFiles(&m, opts.PreSelectedFiles, opts.ProjectRoot)
	}

	if len(opts.Suggestions) > 0 {
		applySuggestions(&m, opts.Suggestions)
	}

	return programRunner(m)
}

func applyPreSelectedFiles(m *Model, files []string, projectRoot string) {
	allFiles, _ := filepicker.ListProjectFiles(projectRoot)

	allSet := map[string]bool{}
	for _, f := range allFiles {
		allSet[f] = true
	}

	suggested := map[string]bool{}
	var validFiles []string
	for _, f := range files {
		if allSet[f] {
			suggested[f] = true
			validFiles = append(validFiles, f)
		}
	}

	sorted := make([]string, len(validFiles))
	copy(sorted, validFiles)
	sort.Strings(sorted)

	for i, s := range m.steps {
		if s.fieldType == "filepicker" {
			m.steps[i].suggestedFiles = sorted
			m.steps[i].filtered = sorted
			m.steps[i].allFiles = allFiles
			if m.steps[i].selectedMap == nil {
				m.steps[i].selectedMap = map[int]bool{}
			}
			for j, f := range allFiles {
				if suggested[f] {
					m.steps[i].selectedMap[j] = true
				}
			}
			break
		}
	}
}

func applySuggestions(m *Model, suggestions map[string]string) {
	for i, s := range m.steps {
		if val, ok := suggestions[s.name]; ok {
			if s.fieldType == "multiselect" && s.selectedMap != nil {
				parts := strings.SplitSeq(val, ",")
				for part := range parts {
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
