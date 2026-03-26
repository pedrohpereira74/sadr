package wizard

import (
	"testing"
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

