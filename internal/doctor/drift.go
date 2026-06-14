package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Drift is a documentation/contract mismatch the AI detected between the diff
// and a record. ID is assigned deterministically by Go (not the AI) so it is
// stable across CI runs and usable as a ChatOps approval handle.
type Drift struct {
	ID      string `json:"id"`
	Record  string `json:"record"`
	FileRef string `json:"file_ref"`
	Section string `json:"section"`
	Summary string `json:"summary"`
}

// RecordDoc is the documentation view of a record passed to the AI.
type RecordDoc struct {
	ID       string
	FileRef  string
	Sections map[string]string
}

// DriftID is a stable short identifier derived from the record and section.
func DriftID(record, section string) string {
	sum := sha256.Sum256([]byte(record + "\x00" + section))
	return hex.EncodeToString(sum[:])[:8]
}

// BuildDriftPrompt assembles the JSON-only drift-detection prompt: the
// compressed diff, the changed-file skeletons, and the documentation of the
// affected records. The AI must answer with a JSON array of drift objects.
func BuildDriftPrompt(diff string, skeletons map[string]string, docs []RecordDoc) string {
	var b strings.Builder

	b.WriteString("You are a software documentation auditor. Compare the code change against the existing records and detect API/contract DRIFT: places where the documented behavior, signature or contract no longer matches the code.\n\n")
	b.WriteString("CRITICAL RULES:\n")
	b.WriteString("1. Output ONLY a valid raw JSON array. No markdown, no backticks, no prose.\n")
	b.WriteString("2. Each element MUST have exactly these keys: \"record\", \"file_ref\", \"section\", \"summary\".\n")
	b.WriteString("3. \"record\" and \"file_ref\" MUST match one of the records below; \"section\" MUST be one of that record's section names.\n")
	b.WriteString("4. \"summary\" is a one-sentence description of what drifted.\n")
	b.WriteString("5. If there is NO drift, output exactly: []\n\n")

	b.WriteString("=== DIFF (compressed) ===\n")
	b.WriteString(diff)
	b.WriteString("\n\n=== CHANGED FILE SKELETONS ===\n")
	for _, path := range sortedKeys(skeletons) {
		fmt.Fprintf(&b, "--- %s\n%s\n", path, skeletons[path])
	}

	b.WriteString("\n=== RECORDS ===\n")
	for _, d := range docs {
		fmt.Fprintf(&b, "RECORD %s (file_ref: %s)\n", d.ID, d.FileRef)
		for _, name := range sortedKeys(d.Sections) {
			fmt.Fprintf(&b, "  [%s]: %s\n", name, d.Sections[name])
		}
	}

	return b.String()
}

// ParseDrifts parses the AI's JSON array response into drifts, stripping any
// accidental code fences and assigning each a stable ID. An empty array yields
// an empty slice and no error.
func ParseDrifts(raw string) ([]Drift, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	var drifts []Drift
	if err := json.Unmarshal([]byte(s), &drifts); err != nil {
		return nil, fmt.Errorf("failed to parse drift response: %w", err)
	}
	for i := range drifts {
		drifts[i].ID = DriftID(drifts[i].Record, drifts[i].Section)
	}
	return drifts, nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
