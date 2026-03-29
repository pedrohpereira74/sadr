package wizard

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelInitializesWithSnippet(t *testing.T) {
	m := NewModel()
	if m.currentStep != 0 {
		t.Errorf("expected step 0, got %d", m.currentStep)
	}
	if m.steps[0].name != "snippet" {
		t.Errorf("expected first step 'snippet', got '%s'", m.steps[0].name)
	}
}

func TestModelHasAllSteps(t *testing.T) {
	m := NewModel()
	expected := []string{"snippet", "title", "tags", "file_ref"}
	if len(m.steps) != len(expected) {
		t.Fatalf("expected %d steps, got %d", len(expected), len(m.steps))
	}
	for i, name := range expected {
		if m.steps[i].name != name {
			t.Errorf("step %d: expected '%s', got '%s'", i, name, m.steps[i].name)
		}
	}
}

func TestModelCompleted(t *testing.T) {
	m := NewModel()
	m.steps[0].value = "some code"
	m.steps[1].value = "Use retry"
	m.steps[2].value = "api,performance"
	m.steps[3].value = "internal/http/client.go"
	m.currentStep = len(m.steps)

	if !m.Completed() {
		t.Error("expected wizard to be completed")
	}
}

func TestModelResult(t *testing.T) {
	m := NewModel()
	m.steps[0].value = "client := retryablehttp.NewClient()"
	m.steps[1].value = "Use retry"
	m.steps[2].value = "api,performance"
	m.steps[3].value = "N/A"
	m.currentStep = len(m.steps)

	result := m.Result()
	if result["snippet"] != "client := retryablehttp.NewClient()" {
		t.Errorf("expected snippet content, got '%s'", result["snippet"])
	}
	if result["title"] != "Use retry" {
		t.Errorf("expected title 'Use retry', got '%s'", result["title"])
	}
	if result["tags"] != "api,performance" {
		t.Errorf("expected tags 'api,performance', got '%s'", result["tags"])
	}
	if result["file_ref"] != "N/A" {
		t.Errorf("expected file_ref 'N/A', got '%s'", result["file_ref"])
	}
}

func TestModelHasSnippetStep(t *testing.T) {
	m := NewModel()
	found := false
	for _, s := range m.steps {
		if s.name == "snippet" {
			found = true
			if s.fieldType != "editor" {
				t.Errorf("expected snippet type 'editor', got '%s'", s.fieldType)
			}
		}
	}
	if !found {
		t.Error("expected a 'snippet' step in wizard")
	}
}

// --- NewModelFromConfig ---

func TestNewModelFromConfigTextFields(t *testing.T) {
	fields := []FieldDef{
		{Name: "title", Type: "text", Required: true},
		{Name: "context", Type: "multitext", Required: false},
		{Name: "alternatives", Type: "list", Required: false},
	}
	m := NewModelFromConfig(fields)

	// snippet editor + 3 fields
	if len(m.steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(m.steps))
	}
	if m.steps[0].fieldType != "editor" {
		t.Errorf("expected first step to be editor, got '%s'", m.steps[0].fieldType)
	}
	for i := 1; i <= 3; i++ {
		if m.steps[i].fieldType != "text" {
			t.Errorf("step %d: expected 'text', got '%s'", i, m.steps[i].fieldType)
		}
	}
}

func TestNewModelFromConfigSelectField(t *testing.T) {
	fields := []FieldDef{
		{Name: "status", Type: "select", Required: false, Options: []string{"proposed", "accepted"}, Default: "proposed"},
	}
	m := NewModelFromConfig(fields)

	if len(m.steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(m.steps))
	}
	s := m.steps[1]
	if s.fieldType != "select" {
		t.Errorf("expected 'select', got '%s'", s.fieldType)
	}
	if len(s.options) != 2 {
		t.Errorf("expected 2 options, got %d", len(s.options))
	}
	if s.value != "proposed" {
		t.Errorf("expected default 'proposed', got '%s'", s.value)
	}
}

func TestNewModelFromConfigMultiselectField(t *testing.T) {
	fields := []FieldDef{
		{Name: "tags", Type: "multiselect", Required: true, Options: []string{"api", "db", "security"}},
	}
	m := NewModelFromConfig(fields)

	s := m.steps[1]
	if s.fieldType != "multiselect" {
		t.Errorf("expected 'multiselect', got '%s'", s.fieldType)
	}
	if s.selectedMap == nil {
		t.Error("expected selectedMap to be initialized")
	}
	if len(s.options) != 3 {
		t.Errorf("expected 3 options, got %d", len(s.options))
	}
}

func TestNewModelFromConfigFilepathField(t *testing.T) {
	fields := []FieldDef{
		{Name: "file_ref", Type: "filepath", Required: false},
	}
	m := NewModelFromConfig(fields)

	s := m.steps[1]
	if s.fieldType != "text" {
		t.Errorf("expected 'text' for filepath, got '%s'", s.fieldType)
	}
	if !strings.Contains(s.prompt, ":skip") {
		t.Errorf("expected filepath prompt to mention :skip, got '%s'", s.prompt)
	}
}

// --- removeEditorSteps ---

func TestRemoveEditorSteps(t *testing.T) {
	m := NewModel()
	before := len(m.steps)

	m = removeEditorSteps(m)
	for _, s := range m.steps {
		if s.fieldType == "editor" {
			t.Error("found editor step after removal")
		}
	}
	if len(m.steps) >= before {
		t.Errorf("expected fewer steps after removal, got %d (was %d)", len(m.steps), before)
	}
}

// --- applySuggestions ---

func TestApplySuggestionsText(t *testing.T) {
	m := NewModel()
	m = removeEditorSteps(m)

	suggestions := map[string]string{
		"title":    "Suggested title",
		"file_ref": "src/main.go",
	}
	applySuggestions(&m, suggestions)

	if m.steps[0].value != "Suggested title" {
		t.Errorf("expected title 'Suggested title', got '%s'", m.steps[0].value)
	}
	if m.steps[2].value != "src/main.go" {
		t.Errorf("expected file_ref 'src/main.go', got '%s'", m.steps[2].value)
	}
}

func TestApplySuggestionsMultiselect(t *testing.T) {
	m := NewModel()
	m = removeEditorSteps(m)

	suggestions := map[string]string{
		"tags": "api,security",
	}
	applySuggestions(&m, suggestions)

	tagsStep := m.steps[1] // tags is index 1 after removing editor
	if !tagsStep.selectedMap[1] {
		t.Error("expected 'api' (index 1) to be selected")
	}
	if !tagsStep.selectedMap[3] {
		t.Error("expected 'security' (index 3) to be selected")
	}
	if tagsStep.selectedMap[0] {
		t.Error("expected 'architecture' (index 0) NOT to be selected")
	}
}

func TestApplySuggestionsCaseInsensitive(t *testing.T) {
	m := NewModel()
	m = removeEditorSteps(m)

	suggestions := map[string]string{"tags": "API,Security"}
	applySuggestions(&m, suggestions)

	tagsStep := m.steps[1]
	if !tagsStep.selectedMap[1] {
		t.Error("expected 'api' to match 'API' case-insensitively")
	}
	if !tagsStep.selectedMap[3] {
		t.Error("expected 'security' to match 'Security' case-insensitively")
	}
}

// --- Update: CtrlC quits ---

func TestUpdateCtrlCQuits(t *testing.T) {
	m := NewModel()
	m = removeEditorSteps(m)
	m.textarea = initTextarea()

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updated, _ := m.Update(msg)
	final := updated.(Model)

	if !final.quitting {
		t.Error("expected quitting to be true after Ctrl+C")
	}
}

// --- Update: select navigation ---

func TestUpdateSelectNavigation(t *testing.T) {
	fields := []FieldDef{
		{Name: "status", Type: "select", Required: true, Options: []string{"proposed", "accepted", "deprecated"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	if m.steps[0].cursorPos != 0 {
		t.Fatalf("expected initial cursor 0, got %d", m.steps[0].cursorPos)
	}

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.steps[0].cursorPos != 1 {
		t.Errorf("expected cursor 1 after down, got %d", m.steps[0].cursorPos)
	}

	// Move down again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.steps[0].cursorPos != 2 {
		t.Errorf("expected cursor 2 after second down, got %d", m.steps[0].cursorPos)
	}

	// Move down at boundary — should stay
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.steps[0].cursorPos != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m.steps[0].cursorPos)
	}

	// Move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.steps[0].cursorPos != 1 {
		t.Errorf("expected cursor 1 after up, got %d", m.steps[0].cursorPos)
	}
}

func TestUpdateSelectEnterConfirms(t *testing.T) {
	fields := []FieldDef{
		{Name: "status", Type: "select", Required: true, Options: []string{"proposed", "accepted"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	// Move down to "accepted"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Enter to confirm
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.steps[0].value != "accepted" {
		t.Errorf("expected 'accepted', got '%s'", m.steps[0].value)
	}
	if m.currentStep != 1 {
		t.Errorf("expected to advance to step 1, got %d", m.currentStep)
	}
}

// --- Update: multiselect ---

func TestUpdateMultiselectToggle(t *testing.T) {
	fields := []FieldDef{
		{Name: "tags", Type: "multiselect", Required: true, Options: []string{"api", "db", "security"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	// Toggle first option (space)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	if !m.steps[0].selectedMap[0] {
		t.Error("expected option 0 to be selected after space")
	}

	// Toggle again to deselect
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	if m.steps[0].selectedMap[0] {
		t.Error("expected option 0 to be deselected after second space")
	}
}

func TestUpdateMultiselectEnterConfirms(t *testing.T) {
	fields := []FieldDef{
		{Name: "tags", Type: "multiselect", Required: false, Options: []string{"api", "db", "security"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	// Select "api" (index 0)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)

	// Move to "security" (index 2) and select
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)

	// Confirm
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.steps[0].value != "api,security" {
		t.Errorf("expected 'api,security', got '%s'", m.steps[0].value)
	}
}

// --- Update: Tab skips optional ---

func TestUpdateTabSkipsOptional(t *testing.T) {
	fields := []FieldDef{
		{Name: "context", Type: "text", Required: false},
		{Name: "title", Type: "text", Required: true},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	// Tab on optional field
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if m.steps[0].value != "N/A" {
		t.Errorf("expected 'N/A' after tab skip, got '%s'", m.steps[0].value)
	}
	if m.currentStep != 1 {
		t.Errorf("expected to advance to step 1, got %d", m.currentStep)
	}
}

func TestUpdateTabDoesNotSkipRequired(t *testing.T) {
	fields := []FieldDef{
		{Name: "title", Type: "text", Required: true},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if m.currentStep != 0 {
		t.Errorf("expected to stay at step 0 for required field, got %d", m.currentStep)
	}
}

// --- Update: text enter with empty required ---

func TestUpdateTextEnterEmptyRequired(t *testing.T) {
	fields := []FieldDef{
		{Name: "title", Type: "text", Required: true},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	// Enter with empty textarea should not advance
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.currentStep != 0 {
		t.Errorf("expected to stay at step 0 with empty required text, got %d", m.currentStep)
	}
}

// --- Update: editorFinishedMsg ---

func TestUpdateEditorFinishedAdvances(t *testing.T) {
	m := NewModel()
	m.textarea = initTextarea()

	msg := editorFinishedMsg{content: "some snippet code"}
	updated, _ := m.Update(msg)
	final := updated.(Model)

	if final.steps[0].value != "some snippet code" {
		t.Errorf("expected snippet value, got '%s'", final.steps[0].value)
	}
	if final.currentStep != 1 {
		t.Errorf("expected to advance to step 1, got %d", final.currentStep)
	}
}

// --- Update: WindowSizeMsg ---

func TestUpdateWindowSizeMsg(t *testing.T) {
	m := NewModel()
	m = removeEditorSteps(m)
	m.textarea = initTextarea()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	final := updated.(Model)

	if final.width != 120 {
		t.Errorf("expected width 120, got %d", final.width)
	}
}

// --- Update: completed model quits ---

func TestUpdateCompletedModelQuits(t *testing.T) {
	m := NewModel()
	m.currentStep = len(m.steps)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected Quit command for completed model")
	}
}

// --- View ---

func TestViewQuitting(t *testing.T) {
	m := NewModel()
	m.quitting = true

	view := m.View()
	if view != "" {
		t.Errorf("expected empty view on quit, got '%s'", view)
	}
}

func TestViewCompleted(t *testing.T) {
	m := NewModel()
	m.currentStep = len(m.steps)

	view := m.View()
	if !strings.Contains(view, "Done") {
		t.Errorf("expected 'Done' in view, got '%s'", view)
	}
}

func TestViewEditorStep(t *testing.T) {
	m := NewModel()
	view := m.View()
	if !strings.Contains(view, "Press Enter to open editor") {
		t.Errorf("expected editor prompt in view, got '%s'", view)
	}
}

func TestViewSelectStep(t *testing.T) {
	fields := []FieldDef{
		{Name: "status", Type: "select", Required: true, Options: []string{"proposed", "accepted"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	view := m.View()
	if !strings.Contains(view, "proposed") {
		t.Errorf("expected 'proposed' in view, got '%s'", view)
	}
	if !strings.Contains(view, "> ") {
		t.Errorf("expected cursor '> ' in view, got '%s'", view)
	}
}

func TestViewMultiselectStep(t *testing.T) {
	fields := []FieldDef{
		{Name: "tags", Type: "multiselect", Required: true, Options: []string{"api", "db"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	view := m.View()
	if !strings.Contains(view, "[ ]") {
		t.Errorf("expected unchecked checkbox in view, got '%s'", view)
	}
	if !strings.Contains(view, "space to toggle") {
		t.Errorf("expected multiselect instructions in view, got '%s'", view)
	}
}

func TestViewMultiselectChecked(t *testing.T) {
	fields := []FieldDef{
		{Name: "tags", Type: "multiselect", Required: true, Options: []string{"api", "db"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)
	m.steps[0].selectedMap[0] = true

	view := m.View()
	if !strings.Contains(view, "[x]") {
		t.Errorf("expected checked checkbox in view, got '%s'", view)
	}
}

func TestViewTextStep(t *testing.T) {
	fields := []FieldDef{
		{Name: "title", Type: "text", Required: true},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	view := m.View()
	if !strings.Contains(view, "title") {
		t.Errorf("expected 'title' in view, got '%s'", view)
	}
	if !strings.Contains(view, "(required)") {
		t.Errorf("expected '(required)' hint in view, got '%s'", view)
	}
}

func TestViewOptionalHint(t *testing.T) {
	fields := []FieldDef{
		{Name: "context", Type: "text", Required: false},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	view := m.View()
	if !strings.Contains(view, "(tab to skip)") {
		t.Errorf("expected '(tab to skip)' hint in view, got '%s'", view)
	}
}

func TestRunWithFieldsAndSuggestions(t *testing.T) {
	old := programRunner
	programRunner = func(m Model) (map[string]string, error) { return m.Result(), nil }
	defer func() { programRunner = old }()

	fields := []FieldDef{
		{Name: "title", Type: "text", Required: true},
		{Name: "tags", Type: "multiselect", Required: true, Options: []string{"api", "db"}},
	}
	suggestions := map[string]string{
		"title": "AI suggested title",
		"tags":  "api",
	}

	result, err := Run(Options{
		SkipEditor:  true,
		Fields:      fields,
		Suggestions: suggestions,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["title"] != "AI suggested title" {
		t.Errorf("expected suggested title, got '%s'", result["title"])
	}
}

func TestRunSkipEditor(t *testing.T) {
	old := programRunner
	programRunner = func(m Model) (map[string]string, error) { return m.Result(), nil }
	defer func() { programRunner = old }()

	result, err := Run(Options{SkipEditor: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["snippet"]; ok {
		t.Error("expected no snippet step when SkipEditor is true")
	}
	if _, ok := result["title"]; !ok {
		t.Error("expected title step to be present")
	}
}

func TestRunDefaultModel(t *testing.T) {
	old := programRunner
	programRunner = func(m Model) (map[string]string, error) { return m.Result(), nil }
	defer func() { programRunner = old }()

	result, err := Run(Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["snippet"]; !ok {
		t.Error("expected snippet key in result")
	}
	if _, ok := result["title"]; !ok {
		t.Error("expected title key in result")
	}
}

// --- Update: KeyEnter on editor step ---

func TestUpdateKeyEnterOnEditorSpawnsProcess(t *testing.T) {
	m := NewModel()
	m.textarea = initTextarea()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil command when pressing Enter on editor step")
	}
}

func TestUpdateKeyEnterOnEditorSetsTempFile(t *testing.T) {
	m := NewModel()
	m.textarea = initTextarea()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(Model)

	if final.tempFile == "" {
		t.Error("expected tempFile to be set after pressing Enter on editor step")
	}
	if final.tempFile != "" {
		t.Cleanup(func() { _ = os.Remove(final.tempFile) })
	}
}

// --- Update: default case space rune toggles multiselect ---

func TestUpdateDefaultSpaceRuneTogglesMultiselect(t *testing.T) {
	fields := []FieldDef{
		{Name: "tags", Type: "multiselect", Required: true, Options: []string{"api", "db", "security"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	spaceRune := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	updated, _ := m.Update(spaceRune)
	m = updated.(Model)

	if !m.steps[0].selectedMap[0] {
		t.Error("expected option 0 to be selected after space rune via default case")
	}

	// Toggle off
	updated, _ = m.Update(spaceRune)
	m = updated.(Model)

	if m.steps[0].selectedMap[0] {
		t.Error("expected option 0 to be deselected after second space rune")
	}
}

// --- Init ---

func TestInitReturnsNil(t *testing.T) {
	m := NewModel()
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil")
	}
}
