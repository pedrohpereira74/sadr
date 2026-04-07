package compress

import (
	"regexp"
	"strings"
)

var (
	reLineComments  = regexp.MustCompile(`(?m)^\s*(//|#).*$`)
	reBlockComments = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reEmptyLines    = regexp.MustCompile(`(?m)^\s*$\n`)
	reIndentation   = regexp.MustCompile(`[ \t]+`)
)

// ZipSourceCode strips comments, empty lines and normalizes whitespace.
func ZipSourceCode(raw string) string {
	s := strings.ReplaceAll(raw, "\r", "")
	s = reLineComments.ReplaceAllString(s, "")
	s = reBlockComments.ReplaceAllString(s, "")
	s = reEmptyLines.ReplaceAllString(s, "")
	s = reIndentation.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// ZipSnippet compresses source code or strips diff noise from a unified diff.
func ZipSnippet(raw string) string {
	s := ZipSourceCode(raw)
	if !isDiff(s) {
		return s
	}
	var out []string
	for line := range strings.SplitSeq(s, "\n") {
		if strings.HasPrefix(line, "diff ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				out = append(out, "--- "+strings.TrimPrefix(parts[len(parts)-1], "b/"))
			}
			continue
		}
		if strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			out = append(out, line)
			continue
		}
	}
	return strings.Join(out, "\n")
}

func isDiff(s string) bool {
	return strings.HasPrefix(s, "diff ") || strings.Contains(s, "\ndiff ")
}
