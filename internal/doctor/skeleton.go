// Package doctor holds the pure logic for the `sadr doctor` CI gatekeeper:
// source skeletons, deterministic record validation and diff filtering. It has
// no I/O side effects and no dependency on git, the AI or GitHub.
package doctor

import (
	"strings"

	"github.com/pedrohpereira74/sadr/internal/compress"
)

// declKeywords are line-leading tokens that mark a declaration across the
// common languages. Best-effort and language-agnostic — no AST/tree-sitter.
var declKeywords = []string{
	"func", "type", "interface", "struct", "const", "var", "package", "import",
	"class", "def", "function", "export", "public", "private", "protected",
	"static", "enum", "trait", "impl", "fn", "module", "abstract",
}

// Skeleton reduces source code to its declaration lines — the signatures of
// functions, types and classes — dropping bodies, comments and blank lines so
// the AI sees the API surface at a fraction of the token cost. It first runs
// compress.ZipSourceCode, then keeps only lines that look like declarations.
func Skeleton(raw string) string {
	cleaned := compress.ZipSourceCode(raw)
	var out []string
	for line := range strings.SplitSeq(cleaned, "\n") {
		if isDeclarationLine(line) {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return strings.Join(out, "\n")
}

func isDeclarationLine(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return false
	}
	for _, kw := range declKeywords {
		if t == kw || strings.HasPrefix(t, kw+" ") || strings.HasPrefix(t, kw+"(") {
			return true
		}
	}
	return false
}
