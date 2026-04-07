package sourcecode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/enricher"
	"github.com/pedrohpereira74/sadr/internal/model"
)

func TestSourceCodeEnricher(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "src"), 0755)
	_ = os.WriteFile(filepath.Join(root, "src", "auth.go"), []byte("package auth\n\nfunc Login() {}"), 0644)

	record := model.Record{FileRef: "src/auth.go", Fields: map[string]string{}}
	e := Enricher{}
	ctx := e.Enrich(enricher.RecordContext{}, record, root)

	if len(ctx.SourceFiles) != 1 {
		t.Fatalf("expected 1 source file, got %d", len(ctx.SourceFiles))
	}
	if ctx.SourceFiles[0].SourcePath != "src/auth.go" {
		t.Errorf("expected src/auth.go, got %s", ctx.SourceFiles[0].SourcePath)
	}
	if ctx.SourceFiles[0].SourceCode == "" {
		t.Error("expected source code to be populated")
	}
}

func TestSourceCodeEnricherMultipleFiles(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "src"), 0755)
	_ = os.WriteFile(filepath.Join(root, "src", "auth.go"), []byte("package auth"), 0644)
	_ = os.WriteFile(filepath.Join(root, "src", "db.go"), []byte("package db"), 0644)

	record := model.Record{FileRef: "src/auth.go,src/db.go", Fields: map[string]string{}}
	e := Enricher{}
	ctx := e.Enrich(enricher.RecordContext{}, record, root)

	if len(ctx.SourceFiles) != 2 {
		t.Fatalf("expected 2 source files, got %d", len(ctx.SourceFiles))
	}
}

func TestSourceCodeEnricherNoFileRef(t *testing.T) {
	e := Enricher{}
	record := model.Record{FileRef: model.NoFileRef, Fields: map[string]string{}}
	ctx := e.Enrich(enricher.RecordContext{}, record, "/tmp")
	if len(ctx.SourceFiles) != 0 {
		t.Error("expected no source files for N/A file_ref")
	}
}

func TestSourceCodeEnricherWithTestFile(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "src"), 0755)
	_ = os.WriteFile(filepath.Join(root, "src", "auth.go"), []byte("package auth"), 0644)
	_ = os.WriteFile(filepath.Join(root, "src", "auth_test.go"), []byte("package auth\nfunc TestAuth(t *testing.T){}"), 0644)

	record := model.Record{FileRef: "src/auth.go", Fields: map[string]string{}}
	e := Enricher{}
	ctx := e.Enrich(enricher.RecordContext{}, record, root)

	if len(ctx.SourceFiles) != 1 {
		t.Fatalf("expected 1 source file, got %d", len(ctx.SourceFiles))
	}
	if ctx.SourceFiles[0].TestPath != "src/auth_test.go" {
		t.Errorf("expected test file, got %s", ctx.SourceFiles[0].TestPath)
	}
}

func TestFindTestFileGo(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "internal", "auth"), 0755)
	_ = os.WriteFile(filepath.Join(root, "internal", "auth", "auth_test.go"), []byte(""), 0644)

	result := FindTestFile(root, "internal/auth/auth.go")
	if result != "internal/auth/auth_test.go" {
		t.Errorf("expected internal/auth/auth_test.go, got %s", result)
	}
}

func TestFindTestFileNotFound(t *testing.T) {
	root := t.TempDir()
	result := FindTestFile(root, "src/main.go")
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestSourceCodeEnricherAbsolutePathIgnored(t *testing.T) {
	root := t.TempDir()
	record := model.Record{FileRef: "/etc/passwd", Fields: map[string]string{}}
	e := Enricher{}
	ctx := e.Enrich(enricher.RecordContext{}, record, root)
	if len(ctx.SourceFiles) != 0 {
		t.Errorf("expected absolute path to be ignored, got %d source files", len(ctx.SourceFiles))
	}
}

func TestSourceCodeEnricherPathTraversalIgnored(t *testing.T) {
	root := t.TempDir()
	record := model.Record{FileRef: "../secret.go", Fields: map[string]string{}}
	e := Enricher{}
	ctx := e.Enrich(enricher.RecordContext{}, record, root)
	if len(ctx.SourceFiles) != 0 {
		t.Errorf("expected path traversal to be ignored, got %d source files", len(ctx.SourceFiles))
	}
}
