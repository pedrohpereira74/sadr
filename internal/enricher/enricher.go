package enricher

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pedrohpereira74/sadr/internal/config"
	"github.com/pedrohpereira74/sadr/internal/jira"
	"github.com/pedrohpereira74/sadr/internal/model"
)

var (
	reLineComments  = regexp.MustCompile(`(?m)^\s*(//|#).*$`)
	reBlockComments = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reEmptyLines    = regexp.MustCompile(`(?m)^\s*$\n`)
	reIndentation   = regexp.MustCompile(`[ \t]+`)
)

func ZipSourceCode(raw string) string {
	s := strings.ReplaceAll(raw, "\r", "")
	s = reLineComments.ReplaceAllString(s, "")
	s = reBlockComments.ReplaceAllString(s, "")
	s = reEmptyLines.ReplaceAllString(s, "")
	s = reIndentation.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

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

type RecordContext struct {
	RecordTitle   string
	RecordType    string
	RecordTags    string
	RecordSnippet string
	RecordFields  map[string]string
	SourceFiles []SourceFile
	JiraIssue *JiraIssueData
}

type SourceFile struct {
	SourceCode string
	SourcePath string
	TestCode   string
	TestPath   string
}

type JiraIssueData struct {
	Key         string
	Summary     string
	Status      string
	Description string
	Assignee    string
}

type Enricher interface {
	Name() string
	Enrich(ctx RecordContext, record model.Record, projectRoot string) RecordContext
}

func BuildContext(record model.Record, enrichers []Enricher, projectRoot string) RecordContext {
	ctx := RecordContext{
		RecordTitle:   record.Title,
		RecordType:    record.Type,
		RecordTags:    record.Fields["tags"],
		RecordSnippet: record.Snippet,
		RecordFields:  record.Fields,
	}

	for _, e := range enrichers {
		ctx = e.Enrich(ctx, record, projectRoot)
	}

	return ctx
}

const maxFileBytes = 32 * 1024

type SourceCodeEnricher struct{}

func (e SourceCodeEnricher) Name() string { return "source_code" }

func (e SourceCodeEnricher) Enrich(ctx RecordContext, record model.Record, projectRoot string) RecordContext {
	if record.FileRef == "" || record.FileRef == model.NoFileRef {
		return ctx
	}

	for p := range strings.SplitSeq(record.FileRef, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		fullPath := filepath.Join(projectRoot, p)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		if len(data) > maxFileBytes {
			data = data[:maxFileBytes]
		}

		sf := SourceFile{
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

type JiraEnricher struct {
	client *jira.Client
}

func NewJiraEnricher(cfg config.JiraConfig) *JiraEnricher {
	if cfg.URL == "" {
		return nil
	}
	token := cfg.APIToken
	if token == "" && cfg.APITokenEnv != "" {
		token = os.Getenv(cfg.APITokenEnv)
	}
	if token == "" {
		return nil
	}
	return &JiraEnricher{
		client: &jira.Client{
			BaseURL:  cfg.URL,
			Email:    cfg.Email,
			APIToken: token,
			HTTP:     &http.Client{Timeout: 10 * time.Second},
		},
	}
}

func (e *JiraEnricher) Name() string { return "jira" }

var jiraKeyPattern = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)

func (e *JiraEnricher) Enrich(rctx RecordContext, record model.Record, _ string) RecordContext {
	parts := []string{record.Title}
	for _, v := range record.Fields {
		parts = append(parts, v)
	}
	keys := jiraKeyPattern.FindAllString(strings.Join(parts, " "), -1)
	if len(keys) == 0 {
		return rctx
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	issues, err := e.client.FetchAll(ctx, keys)
	if err != nil {
		return rctx
	}

	for _, k := range keys {
		if issue, ok := issues[k]; ok {
			rctx.JiraIssue = &JiraIssueData{
				Key:         issue.Key,
				Summary:     issue.Summary,
				Status:      issue.Status,
				Description: issue.Description,
				Assignee:    issue.Assignee,
			}
			break
		}
	}

	return rctx
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
