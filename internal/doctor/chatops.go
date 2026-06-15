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

func RenderComment(conflicts []Collision) string {
	var b strings.Builder
	b.WriteString(CommentMarker)
	b.WriteString("\n## sadr doctor — conflicting records\n\n")
	b.WriteString("These files are documented by more than one active record. ")
	b.WriteString("If an older decision no longer applies, deprecate it:\n\n")
	for _, c := range conflicts {
		fmt.Fprintf(&b, "- `%s` — %s\n", c.FileRef, strings.Join(c.Records, ", "))
	}
	b.WriteString("\nReply `/doctor apply <record-ids>` (comma-separated, e.g. `alice/1`) ")
	b.WriteString("to mark those records as deprecated. Unresolved conflicts block the merge.\n")
	return b.String()
}
