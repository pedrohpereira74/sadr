package doctor

import (
	"fmt"
	"strings"
)

// CommentMarker identifies the doctor comment on a PR so it can be upserted
// (found and updated) across CI runs instead of duplicated.
const CommentMarker = "<!-- sadr-doctor -->"

// ParseApplyCommand parses a PR comment body for a `/doctor apply` ChatOps
// command. The command must start a line (matching the GitHub slash-command
// convention; avoids matching commands quoted inside prose). It returns the
// requested drift ids, whether "all" was requested, and whether a valid command
// was found.
func ParseApplyCommand(body string) (ids []string, all bool, ok bool) {
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		const prefix = "/doctor apply"
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if rest == "" {
			return nil, false, false
		}
		if rest == "all" {
			return nil, true, true
		}
		for part := range strings.SplitSeq(rest, ",") {
			if part = strings.TrimSpace(part); part != "" {
				ids = append(ids, part)
			}
		}
		return ids, false, len(ids) > 0
	}
	return nil, false, false
}

// IsAuthorized reports whether a commenter's GitHub author_association is
// allowed to trigger an apply. This guards the issue_comment workflow against
// privilege escalation by arbitrary commenters.
func IsAuthorized(association string) bool {
	switch strings.ToUpper(strings.TrimSpace(association)) {
	case "OWNER", "MEMBER", "COLLABORATOR":
		return true
	}
	return false
}

// SelectDrifts returns the drifts approved by id (or all of them).
func SelectDrifts(drifts []Drift, ids []string, all bool) []Drift {
	if all {
		return drifts
	}
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []Drift
	for _, d := range drifts {
		if want[d.ID] {
			out = append(out, d)
		}
	}
	return out
}

// RenderComment renders the PR comment listing the detected drifts, prefixed
// with CommentMarker for idempotent upserts.
func RenderComment(drifts []Drift) string {
	var b strings.Builder
	b.WriteString(CommentMarker)
	b.WriteString("\n## sadr doctor — documentation drift detected\n\n")
	b.WriteString("These records look out of date with this change:\n\n")
	for i, d := range drifts {
		fmt.Fprintf(&b, "%d. `%s` — **%s** §%s — %s\n", i+1, d.ID, d.Record, d.Section, d.Summary)
	}
	b.WriteString("\nReply `/doctor apply <ids>` (comma-separated drift ids) or `/doctor apply all` ")
	b.WriteString("to let sadr rewrite the approved sections. Unresolved drift blocks the merge.\n")
	return b.String()
}
