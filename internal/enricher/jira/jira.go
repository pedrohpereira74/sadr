package jira

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/pedrohpereira74/sadr/internal/enricher"
	jiraclient "github.com/pedrohpereira74/sadr/internal/jira"
	"github.com/pedrohpereira74/sadr/internal/model"
)

var jiraKeyPattern = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)

type Enricher struct {
	client *jiraclient.Client
}

func New(baseURL string, cfg jiraclient.ClientConfig) *Enricher {
	if baseURL == "" {
		return nil
	}
	cfg.BaseURL = baseURL
	c := jiraclient.NewClientFromConfig(cfg)
	if c.Username == "" && c.Token == "" && c.ConsumerKey == "" {
		return nil
	}
	return &Enricher{client: c}
}

func (e *Enricher) Name() string { return "jira" }

func (e *Enricher) Enrich(rctx enricher.RecordContext, record model.Record, _ string) enricher.RecordContext {
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
			rctx.JiraIssue = &enricher.JiraIssueData{
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
