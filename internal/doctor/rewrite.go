package doctor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RewriteRequest asks the AI to bring one record section back in line with the
// code change.
type RewriteRequest struct {
	Record  string
	Section string
	Current string
	Summary string
}

// Rewrite is the AI's new content for a single record section.
type Rewrite struct {
	Record  string `json:"record"`
	Section string `json:"section"`
	Content string `json:"content"`
}

// BuildRewritePrompt assembles the JSON-only prompt asking the AI to rewrite the
// approved sections so they match the code change.
func BuildRewritePrompt(diff string, reqs []RewriteRequest) string {
	var b strings.Builder

	b.WriteString("You update architecture decision records so their documentation matches a code change.\n\n")
	b.WriteString("CRITICAL RULES:\n")
	b.WriteString("1. Output ONLY a valid raw JSON array. No markdown, no backticks, no prose.\n")
	b.WriteString("2. Each element MUST have exactly these keys: \"record\", \"section\", \"content\".\n")
	b.WriteString("3. Echo \"record\" and \"section\" exactly as given; put the rewritten section text in \"content\".\n")
	b.WriteString("4. Rewrite only what the change requires; keep the original tone and stay factual.\n\n")

	b.WriteString("=== DIFF (compressed) ===\n")
	b.WriteString(diff)
	b.WriteString("\n\n=== SECTIONS TO REWRITE ===\n")
	for _, r := range reqs {
		fmt.Fprintf(&b, "RECORD %s | SECTION %s\nDRIFT: %s\nCURRENT:\n%s\n\n", r.Record, r.Section, r.Summary, r.Current)
	}

	return b.String()
}

// ParseRewrites parses the AI's JSON array of rewritten sections, stripping any
// accidental code fences.
func ParseRewrites(raw string) ([]Rewrite, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	var rewrites []Rewrite
	if err := json.Unmarshal([]byte(s), &rewrites); err != nil {
		return nil, fmt.Errorf("failed to parse rewrite response: %w", err)
	}
	return rewrites, nil
}
