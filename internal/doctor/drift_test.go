package doctor

import (
	"strings"
	"testing"
)

func TestDriftIDStable(t *testing.T) {
	a := DriftID("alice/1", "Decision")
	b := DriftID("alice/1", "Decision")
	c := DriftID("alice/1", "Context")
	if a != b {
		t.Errorf("DriftID must be stable: %q != %q", a, b)
	}
	if a == c {
		t.Error("DriftID must differ per section")
	}
	if len(a) != 8 {
		t.Errorf("expected 8-char id, got %q", a)
	}
}

func TestParseDriftsAssignsIDs(t *testing.T) {
	raw := "```json\n" + `[{"record":"alice/1","file_ref":"api.go","section":"Decision","summary":"signature changed"}]` + "\n```"
	drifts, err := ParseDrifts(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drifts) != 1 {
		t.Fatalf("expected 1 drift, got %d", len(drifts))
	}
	if drifts[0].ID != DriftID("alice/1", "Decision") {
		t.Errorf("expected stable id, got %q", drifts[0].ID)
	}
	if drifts[0].Summary != "signature changed" {
		t.Errorf("unexpected summary %q", drifts[0].Summary)
	}
}

func TestParseDriftsEmpty(t *testing.T) {
	for _, in := range []string{"[]", "  []  ", "```json\n[]\n```", ""} {
		drifts, err := ParseDrifts(in)
		if err != nil {
			t.Errorf("ParseDrifts(%q) errored: %v", in, err)
		}
		if len(drifts) != 0 {
			t.Errorf("ParseDrifts(%q) = %v, want empty", in, drifts)
		}
	}
}

func TestParseDriftsInvalid(t *testing.T) {
	if _, err := ParseDrifts("not json"); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestBuildDriftPromptDeterministicAndComplete(t *testing.T) {
	diff := "@@\n-func Old()\n+func New()"
	skeletons := map[string]string{"api.go": "func New()"}
	docs := []RecordDoc{{
		ID:       "alice/1",
		FileRef:  "api.go",
		Sections: map[string]string{"Decision": "uses Old()", "Context": "legacy"},
	}}

	p1 := BuildDriftPrompt(diff, skeletons, docs)
	p2 := BuildDriftPrompt(diff, skeletons, docs)
	if p1 != p2 {
		t.Error("BuildDriftPrompt must be deterministic")
	}
	for _, want := range []string{"func New()", "RECORD alice/1", "[Decision]", "[]"} {
		if !strings.Contains(p1, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	// Context must come before Decision (sorted section order).
	if strings.Index(p1, "[Context]") > strings.Index(p1, "[Decision]") {
		t.Error("sections should be in sorted order")
	}
}
