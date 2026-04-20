package compress

import (
	"strings"
	"testing"
)

// --- ZipSourceCode ---

func TestZipSourceCodeStripsLineComments(t *testing.T) {
	input := "func foo() {\n// this is a comment\nreturn nil\n}"
	got := ZipSourceCode(input)
	if strings.Contains(got, "this is a comment") {
		t.Errorf("expected line comment stripped, got %q", got)
	}
	if !strings.Contains(got, "return nil") {
		t.Errorf("expected code preserved, got %q", got)
	}
}

func TestZipSourceCodeStripsHashComments(t *testing.T) {
	input := "x = 1\n# python comment\ny = 2"
	got := ZipSourceCode(input)
	if strings.Contains(got, "python comment") {
		t.Errorf("expected hash comment stripped, got %q", got)
	}
}

func TestZipSourceCodeStripsBlockComments(t *testing.T) {
	input := "func foo() {\n/* block\ncomment */\nreturn nil\n}"
	got := ZipSourceCode(input)
	if strings.Contains(got, "block") || strings.Contains(got, "comment") {
		t.Errorf("expected block comment stripped, got %q", got)
	}
	if !strings.Contains(got, "return nil") {
		t.Errorf("expected code preserved, got %q", got)
	}
}

func TestZipSourceCodeRemovesEmptyLines(t *testing.T) {
	input := "line one\n\n\nline two"
	got := ZipSourceCode(input)
	if strings.Contains(got, "\n\n") {
		t.Errorf("expected empty lines removed, got %q", got)
	}
}

func TestZipSourceCodeNormalizesIndentation(t *testing.T) {
	input := "func foo() {\n\t\treturn nil\n}"
	got := ZipSourceCode(input)
	if strings.Contains(got, "\t\t") {
		t.Errorf("expected indentation normalized, got %q", got)
	}
}

func TestZipSourceCodeNormalizesCRLF(t *testing.T) {
	input := "line one\r\nline two\r\n"
	got := ZipSourceCode(input)
	if strings.Contains(got, "\r") {
		t.Errorf("expected CRLF normalized, got %q", got)
	}
}

func TestZipSourceCodeEmptyInput(t *testing.T) {
	if got := ZipSourceCode(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

// --- ZipSnippet ---

func TestZipSnippetPlainCode(t *testing.T) {
	input := "func foo() {\nreturn nil\n}"
	got := ZipSnippet(input)
	if !strings.Contains(got, "return nil") {
		t.Errorf("expected code preserved for non-diff input, got %q", got)
	}
}

func TestZipSnippetDiffStripsIndexAndPlusPlusPlus(t *testing.T) {
	input := "diff --git a/foo.go b/foo.go\nindex abc..def 100644\n--- a/foo.go\n+++ b/foo.go\n@@ -1,3 +1,4 @@\n+added line\n context line"
	got := ZipSnippet(input)
	if strings.Contains(got, "index abc") {
		t.Errorf("expected 'index' line stripped, got %q", got)
	}
	if strings.Contains(got, "+++ b/foo.go") {
		t.Errorf("expected '+++ line' stripped, got %q", got)
	}
	if strings.Contains(got, "--- a/foo.go") {
		t.Errorf("expected '--- line' stripped, got %q", got)
	}
}

func TestZipSnippetDiffKeepsHunkHeader(t *testing.T) {
	input := "diff --git a/foo.go b/foo.go\nindex abc..def 100644\n--- a/foo.go\n+++ b/foo.go\n@@ -1,3 +1,4 @@\n+added line"
	got := ZipSnippet(input)
	if !strings.Contains(got, "@@") {
		t.Errorf("expected hunk header preserved, got %q", got)
	}
}

func TestZipSnippetDiffKeepsAddedAndRemovedLines(t *testing.T) {
	input := "diff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n@@ -1,2 +1,2 @@\n-old line\n+new line"
	got := ZipSnippet(input)
	if !strings.Contains(got, "+new line") {
		t.Errorf("expected added line preserved, got %q", got)
	}
	if !strings.Contains(got, "-old line") {
		t.Errorf("expected removed line preserved, got %q", got)
	}
}

func TestZipSnippetDiffExtractsFilename(t *testing.T) {
	input := "diff --git a/internal/foo.go b/internal/foo.go\n--- a/internal/foo.go\n+++ b/internal/foo.go\n@@ -1 +1 @@\n+x"
	got := ZipSnippet(input)
	if !strings.Contains(got, "--- internal/foo.go") {
		t.Errorf("expected simplified filename header, got %q", got)
	}
}

func TestZipSnippetEmptyInput(t *testing.T) {
	if got := ZipSnippet(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}
