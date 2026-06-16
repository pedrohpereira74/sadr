package wizard

import (
	"os"
	"path/filepath"
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

func TestNewModelFromConfigTextFields(t *testing.T) {
	fields := []FieldDef{
		{Name: "title", Type: "text", Required: true},
		{Name: "context", Type: "text", Required: false},
		{Name: "alternatives", Type: "list", Required: false},
	}
	m := NewModelFromConfig(fields)

	if len(m.steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(m.steps))
	}
	if m.steps[0].fieldType != "editor" {
		t.Errorf("expected first step to be editor, got '%s'", m.steps[0].fieldType)
	}
	if m.steps[1].name != "title" || m.steps[1].fieldType != "text" {
		t.Errorf("expected step 1 to be title/text, got '%s'/'%s'", m.steps[1].name, m.steps[1].fieldType)
	}
	if m.steps[2].fieldType != "filepicker" {
		t.Errorf("expected step 2 to be filepicker, got '%s'", m.steps[2].fieldType)
	}
	if m.steps[3].name != "context" || m.steps[3].fieldType != "text" {
		t.Errorf("expected step 3 to be context/text, got '%s'/'%s'", m.steps[3].name, m.steps[3].fieldType)
	}
	if m.steps[4].name != "alternatives" || m.steps[4].fieldType != "text" {
		t.Errorf("expected step 4 to be alternatives/text, got '%s'/'%s'", m.steps[4].name, m.steps[4].fieldType)
	}
}

func TestNewModelFromConfigSelectField(t *testing.T) {
	fields := []FieldDef{
		{Name: "status", Type: "select", Required: false, Options: []string{"proposed", "accepted"}, Default: "proposed"},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

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

func TestNewModelFileRefIsFilepicker(t *testing.T) {
	m := NewModel()
	last := m.steps[len(m.steps)-1]
	if last.fieldType != "filepicker" {
		t.Errorf("expected last step to be filepicker, got '%s'", last.fieldType)
	}
	if last.name != "file_ref" {
		t.Errorf("expected last step name 'file_ref', got '%s'", last.name)
	}
}

func TestNewModelFromConfigFileRefAfterTitleTags(t *testing.T) {
	fields := []FieldDef{
		{Name: "title", Type: "text", Required: true},
		{Name: "context", Type: "text", Required: false},
	}
	m := NewModelFromConfig(fields)

	if m.steps[2].fieldType != "filepicker" {
		t.Errorf("expected file_ref after title, got '%s'", m.steps[2].fieldType)
	}
	if m.steps[2].name != "file_ref" {
		t.Errorf("expected step name 'file_ref', got '%s'", m.steps[2].name)
	}
	if m.steps[3].name != "context" {
		t.Errorf("expected context to be last, got '%s'", m.steps[3].name)
	}
}

func TestRemoveFileRefSteps(t *testing.T) {
	m := NewModel()
	before := len(m.steps)

	m = removeFileRefSteps(m)
	for _, s := range m.steps {
		if s.fieldType == "filepicker" {
			t.Error("found filepicker step after removal")
		}
	}
	if len(m.steps) >= before {
		t.Errorf("expected fewer steps after removal, got %d (was %d)", len(m.steps), before)
	}
}

func TestFilepickerSkipCommand(t *testing.T) {
	m := NewModel()
	m = removeEditorSteps(m)
	m.textarea = initTextarea()

	fpIdx := 0
	for i := range m.steps {
		if m.steps[i].fieldType == "filepicker" {
			m.currentStep = i
			fpIdx = i
			break
		}
	}

	for _, r := range ":skip" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(Model)

	if final.steps[fpIdx].value != "N/A" {
		t.Errorf("expected 'N/A' after :skip on filepicker, got '%s'", final.steps[fpIdx].value)
	}
}

func TestFilepickerTabDoesNotSkip(t *testing.T) {
	m := NewModel()
	m = removeEditorSteps(m)
	m.textarea = initTextarea()

	fpIdx := 0
	for i := range m.steps {
		if m.steps[i].fieldType == "filepicker" {
			m.currentStep = i
			fpIdx = i
			break
		}
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	final := updated.(Model)

	if final.currentStep != fpIdx {
		t.Errorf("expected to stay at filepicker step (required), but advanced to %d", final.currentStep)
	}
}

func TestFilepickerMultiSelect(t *testing.T) {
	m := NewModel()
	m = removeEditorSteps(m)
	m.textarea = initTextarea()

	fpIdx := 0
	for i := range m.steps {
		if m.steps[i].fieldType == "filepicker" {
			m.currentStep = i
			fpIdx = i
			break
		}
	}

	m.steps[fpIdx].allFiles = []string{"src/a.go", "src/b.go", "src/c.go"}
	m.steps[fpIdx].filtered = []string{"src/a.go", "src/b.go", "src/c.go"}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(Model)

	if final.steps[fpIdx].value != "src/a.go,src/b.go" {
		t.Errorf("expected 'src/a.go,src/b.go', got '%s'", final.steps[fpIdx].value)
	}
}

func TestFilepickerPreSelected(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"handler.go", "model.go", "main.go", "utils.go"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	m := NewModel()
	m = removeEditorSteps(m)
	m.textarea = initTextarea()

	preFiles := []string{"handler.go", "model.go"}
	applyPreSelectedFiles(&m, preFiles, dir)

	fpIdx := 0
	for i := range m.steps {
		if m.steps[i].fieldType == "filepicker" {
			fpIdx = i
			break
		}
	}

	if len(m.steps[fpIdx].suggestedFiles) != 2 {
		t.Fatalf("expected 2 suggested files, got %d", len(m.steps[fpIdx].suggestedFiles))
	}
	if len(m.steps[fpIdx].allFiles) == 0 {
		t.Fatal("expected allFiles to be populated eagerly")
	}

	allIdx := map[string]int{}
	for i, f := range m.steps[fpIdx].allFiles {
		allIdx[f] = i
	}

	if !m.steps[fpIdx].selectedMap[allIdx["handler.go"]] {
		t.Error("expected handler.go to be selected")
	}
	if !m.steps[fpIdx].selectedMap[allIdx["model.go"]] {
		t.Error("expected model.go to be selected")
	}
	if m.steps[fpIdx].selectedMap[allIdx["main.go"]] {
		t.Error("expected main.go to NOT be selected")
	}
	if m.steps[fpIdx].selectedMap[allIdx["utils.go"]] {
		t.Error("expected utils.go to NOT be selected")
	}
}

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

	tagsStep := m.steps[1]
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

func TestUpdateSelectNavigation(t *testing.T) {
	fields := []FieldDef{
		{Name: "status", Type: "select", Required: true, Options: []string{"proposed", "accepted", "deprecated"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)
	m = removeFileRefSteps(m)

	if m.steps[0].cursorPos != 0 {
		t.Fatalf("expected initial cursor 0, got %d", m.steps[0].cursorPos)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.steps[0].cursorPos != 1 {
		t.Errorf("expected cursor 1 after down, got %d", m.steps[0].cursorPos)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.steps[0].cursorPos != 2 {
		t.Errorf("expected cursor 2 after second down, got %d", m.steps[0].cursorPos)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.steps[0].cursorPos != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m.steps[0].cursorPos)
	}

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
	m = removeFileRefSteps(m)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.steps[0].value != "accepted" {
		t.Errorf("expected 'accepted', got '%s'", m.steps[0].value)
	}
	if m.currentStep != 1 {
		t.Errorf("expected to advance to step 1, got %d", m.currentStep)
	}
}

func TestUpdateMultiselectToggle(t *testing.T) {
	fields := []FieldDef{
		{Name: "tags", Type: "multiselect", Required: true, Options: []string{"api", "db", "security"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	if !m.steps[0].selectedMap[0] {
		t.Error("expected option 0 to be selected after space")
	}

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

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.steps[0].value != "api,security" {
		t.Errorf("expected 'api,security', got '%s'", m.steps[0].value)
	}
}

func TestUpdateTabSkipsOptional(t *testing.T) {
	fields := []FieldDef{
		{Name: "context", Type: "text", Required: false},
		{Name: "title", Type: "text", Required: true},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)
	m = removeFileRefSteps(m)

	m.currentStep = 1
	m.textarea = initTextarea()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if m.steps[1].value != "N/A" {
		t.Errorf("expected 'N/A' after tab skip, got '%s'", m.steps[1].value)
	}
	if m.currentStep != 2 {
		t.Errorf("expected to advance to step 2, got %d", m.currentStep)
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

func TestUpdateTextEnterEmptyRequired(t *testing.T) {
	fields := []FieldDef{
		{Name: "title", Type: "text", Required: true},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.currentStep != 0 {
		t.Errorf("expected to stay at step 0 with empty required text, got %d", m.currentStep)
	}
}

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

func TestUpdateCompletedModelQuits(t *testing.T) {
	m := NewModel()
	m.currentStep = len(m.steps)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected Quit command for completed model")
	}
}

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
	if !strings.Contains(view, "enter to open editor") {
		t.Errorf("expected editor prompt in view, got '%s'", view)
	}
}

func TestViewSelectStep(t *testing.T) {
	fields := []FieldDef{
		{Name: "status", Type: "select", Required: true, Options: []string{"proposed", "accepted"}},
	}
	m := NewModelFromConfig(fields)
	m = removeEditorSteps(m)

	for i, s := range m.steps {
		if s.fieldType == "select" {
			m.currentStep = i
			break
		}
	}

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
	if !strings.Contains(view, "space toggle") {
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

	for i, s := range m.steps {
		if s.fieldType == "text" {
			m.currentStep = i
			break
		}
	}

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

	updated, _ = m.Update(spaceRune)
	m = updated.(Model)

	if m.steps[0].selectedMap[0] {
		t.Error("expected option 0 to be deselected after second space rune")
	}
}

func TestInitReturnsBlink(t *testing.T) {
	m := NewModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected Init() to return a blink cmd")
	}
}
