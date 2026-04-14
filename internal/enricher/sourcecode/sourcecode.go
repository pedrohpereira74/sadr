package sourcecode

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/enricher"
	"github.com/pedrohpereira74/sadr/internal/model"
)

const maxFileBytes = 32 * 1024

type Enricher struct{}

func (e Enricher) Name() string { return "source_code" }

func (e Enricher) Enrich(ctx enricher.RecordContext, record model.Record, projectRoot string) enricher.RecordContext {
	if record.FileRef == "" || record.FileRef == model.NoFileRef {
		return ctx
	}

	seen := map[string]bool{}
	for p := range strings.SplitSeq(record.FileRef, ",") {
		p = strings.TrimSpace(p)
		if p == "" || filepath.IsAbs(p) || seen[p] {
			continue
		}
		seen[p] = true

		fullPath := filepath.Join(projectRoot, p)
		rel, err := filepath.Rel(projectRoot, fullPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}

		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		if len(data) > maxFileBytes {
			data = data[:maxFileBytes]
		}

		sf := enricher.SourceFile{
			SourceCode: string(data),
			SourcePath: p,
		}

		testPath := FindTestFile(projectRoot, p)
		if testPath != "" {
			testData, err := os.ReadFile(filepath.Join(projectRoot, testPath))
			if err == nil {
				if len(testData) > maxFileBytes {
					testData = testData[:maxFileBytes]
				}
				sf.TestCode = string(testData)
				sf.TestPath = testPath
			}
		}

		ctx.SourceFiles = append(ctx.SourceFiles, sf)
	}

	return ctx
}

func FindTestFile(projectRoot string, sourceFile string) string {
	ext := filepath.Ext(sourceFile)
	dir := filepath.Dir(sourceFile)
	base := strings.TrimSuffix(filepath.Base(sourceFile), ext)

	var candidates []string

	switch ext {
	case ".go":
		candidates = []string{
			filepath.Join(dir, base+"_test.go"),
		}
	case ".py":
		candidates = []string{
			filepath.Join(dir, "test_"+base+".py"),
			filepath.Join(filepath.Dir(dir), "tests", "test_"+base+".py"),
			filepath.Join(dir, base+"_test.py"),
		}
	case ".js":
		candidates = []string{
			filepath.Join(dir, base+".test.js"),
			filepath.Join(dir, base+".spec.js"),
			filepath.Join(dir, "__tests__", base+".test.js"),
		}
	case ".ts":
		candidates = []string{
			filepath.Join(dir, base+".test.ts"),
			filepath.Join(dir, base+".spec.ts"),
			filepath.Join(dir, "__tests__", base+".test.ts"),
		}
	case ".jsx":
		candidates = []string{
			filepath.Join(dir, base+".test.jsx"),
			filepath.Join(dir, base+".spec.jsx"),
			filepath.Join(dir, "__tests__", base+".test.jsx"),
		}
	case ".tsx":
		candidates = []string{
			filepath.Join(dir, base+".test.tsx"),
			filepath.Join(dir, base+".spec.tsx"),
			filepath.Join(dir, "__tests__", base+".test.tsx"),
		}
	case ".java":
		relDir := strings.Replace(dir, "src/main/java", "src/test/java", 1)
		candidates = []string{
			filepath.Join(relDir, base+"Test.java"),
		}
	case ".rs":
		candidates = []string{
			filepath.Join(filepath.Dir(dir), "tests", base+".rs"),
		}
	case ".rb":
		candidates = []string{
			filepath.Join(filepath.Dir(dir), "spec", base+"_spec.rb"),
			filepath.Join(filepath.Dir(dir), "test", "test_"+base+".rb"),
			filepath.Join(dir, base+"_spec.rb"),
		}
	case ".kt":
		relDir := strings.Replace(dir, "src/main/kotlin", "src/test/kotlin", 1)
		candidates = []string{
			filepath.Join(relDir, base+"Test.kt"),
			filepath.Join(relDir, base+"Spec.kt"),
		}
	case ".cs":
		candidates = []string{
			filepath.Join(dir, base+"Tests.cs"),
			filepath.Join(dir, base+"Test.cs"),
			filepath.Join(filepath.Dir(dir), base+".Tests", base+"Tests.cs"),
		}
	case ".php":
		candidates = []string{
			filepath.Join(dir, base+"Test.php"),
			filepath.Join(filepath.Dir(dir), "tests", base+"Test.php"),
			filepath.Join(filepath.Dir(dir), "Tests", base+"Test.php"),
		}
	case ".swift":
		candidates = []string{
			filepath.Join(dir, base+"Tests.swift"),
			filepath.Join(filepath.Dir(dir), base+"Tests", base+"Tests.swift"),
		}
	case ".c", ".cpp", ".cc":
		candidates = []string{
			filepath.Join(dir, "test_"+base+ext),
			filepath.Join(dir, base+"_test"+ext),
			filepath.Join(filepath.Dir(dir), "tests", "test_"+base+ext),
		}
	}

	for _, c := range candidates {
		fullPath := filepath.Join(projectRoot, c)
		if _, err := os.Stat(fullPath); err == nil {
			return c
		}
	}

	return ""
}
