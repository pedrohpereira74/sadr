package doctor

import (
	"strings"
	"testing"
)

func TestBuildRewritePromptIncludesContext(t *testing.T) {
	reqs := []RewriteRequest{
		{Record: "alice/1", Section: "Decision", Current: "uses Old()", Summary: "renamed to New()"},
	}
	p := BuildRewritePrompt("@@ -1 +1 @@\n-Old\n+New", reqs)
	for _, want := range []string{"RECORD alice/1", "SECTION Decision", "uses Old()", "renamed to New()", "JSON array"} {
		if !strings.Contains(p, want) {
			t.Errorf("rewrite prompt missing %q", want)
		}
	}
}

func TestParseRewrites(t *testing.T) {
	raw := "```json\n" + `[{"record":"alice/1","section":"Decision","content":"now uses New()"}]` + "\n```"
	rw, err := ParseRewrites(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rw) != 1 || rw[0].Content != "now uses New()" || rw[0].Section != "Decision" {
		t.Errorf("unexpected rewrites %+v", rw)
	}
}

func TestParseRewritesEmptyAndInvalid(t *testing.T) {
	if rw, err := ParseRewrites("[]"); err != nil || len(rw) != 0 {
		t.Errorf("empty: got %v, %v", rw, err)
	}
	if _, err := ParseRewrites("nope"); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
