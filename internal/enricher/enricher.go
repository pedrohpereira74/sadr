package enricher

import (
	"strings"

	"github.com/pedrohpereira74/sadr/internal/model"
)

type RecordContext struct {
	RecordTitle   string
	RecordType    string
	RecordTags    string
	RecordSnippet string
	RecordFields  map[string]string
	SourceFiles   []SourceFile
	JiraIssue     *JiraIssueData
}

type JiraIssueData struct {
	Key         string
	Summary     string
	Status      string
	Description string
	Assignee    string
}

type SourceFile struct {
	SourceCode string
	SourcePath string
	TestCode   string
	TestPath   string
}

type Enricher interface {
	Name() string
	Enrich(ctx RecordContext, record model.Record, projectRoot string) RecordContext
}

func BuildContext(record model.Record, enrichers []Enricher, projectRoot string) RecordContext {
	ctx := RecordContext{
		RecordTitle:   record.Title,
		RecordType:    record.Type,
		RecordTags:    strings.Join(record.Tags, ","),
		RecordSnippet: record.Snippet,
		RecordFields:  record.Fields,
	}

	for _, e := range enrichers {
		ctx = e.Enrich(ctx, record, projectRoot)
	}

	return ctx
}
