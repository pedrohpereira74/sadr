package doctor

import (
	"fmt"
	"strings"
)

const CommentMarker = "<!-- sadr-doctor -->"

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

func IsAuthorized(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "OWNER", "MEMBER", "COLLABORATOR", "MAINTAINER", "DEVELOPER":
		return true
	}
	return false
}

func RenderComment(res ValidationResult) string {
	var b strings.Builder
	b.WriteString(CommentMarker)
	b.WriteString("\n## sadr doctor — record issues\n\n")
	if len(res.Collisions) > 0 {
		b.WriteString("Files documented by more than one active record that do not reference each other. ")
		b.WriteString("Deprecate the stale one, or add a `related` entry if they coexist on purpose:\n\n")
		for _, c := range res.Collisions {
			fmt.Fprintf(&b, "- `%s` — %s\n", c.FileRef, strings.Join(c.Records, ", "))
		}
		b.WriteString("\n")
	}
	if len(res.Orphans) > 0 {
		b.WriteString("Active records pointing at files that no longer exist. ")
		b.WriteString("Fix the file_ref or deprecate the record:\n\n")
		for _, o := range res.Orphans {
			fmt.Fprintf(&b, "- %s → `%s`\n", o.Record, o.FileRef)
		}
		b.WriteString("\n")
	}
	b.WriteString("Reply `/doctor apply <record-ids>` (comma-separated, e.g. `alice/1`) ")
	b.WriteString("to mark records as deprecated. Unresolved issues block the merge.\n")
	return b.String()
}
