package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync"
)

type Client struct {
	BaseURL  string
	Email    string
	APIToken string
	HTTP     *http.Client
}

type Issue struct {
	Key         string
	Summary     string
	Status      string
	Assignee    string
	Description string
}

type bulkRequest struct {
	IssueIdsOrKeys []string `json:"issueIdsOrKeys"`
	Fields         []string `json:"fields"`
}

type bulkResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string                 `json:"summary"`
			Status      *statusField           `json:"status"`
			Assignee    *assigneeField         `json:"assignee"`
			Description any                    `json:"description"`
		} `json:"fields"`
	} `json:"issues"`
}

type statusField struct {
	Name string `json:"name"`
}

type assigneeField struct {
	DisplayName string `json:"displayName"`
}

func (c *Client) authHeader() string {
	creds := fmt.Sprintf("%s:%s", c.Email, c.APIToken)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

func (c *Client) BulkFetch(ctx context.Context, keys []string) (map[string]Issue, error) {
	if len(keys) == 0 {
		return map[string]Issue{}, nil
	}

	reqBody := bulkRequest{
		IssueIdsOrKeys: keys,
		Fields:         []string{"summary", "status", "assignee", "description"},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/rest/api/3/issue/bulkfetch", strings.TrimRight(c.BaseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Jira API request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Jira API error (%d): %s", resp.StatusCode, string(body))
	}

	var bulkResp bulkResponse
	if err := json.Unmarshal(body, &bulkResp); err != nil {
		return nil, fmt.Errorf("failed to parse Jira response: %v", err)
	}

	result := make(map[string]Issue, len(bulkResp.Issues))
	for _, issue := range bulkResp.Issues {
		i := Issue{
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Description: extractTextFromADF(issue.Fields.Description),
		}
		if issue.Fields.Status != nil {
			i.Status = issue.Fields.Status.Name
		}
		if issue.Fields.Assignee != nil {
			i.Assignee = issue.Fields.Assignee.DisplayName
		}
		result[issue.Key] = i
	}

	return result, nil
}

func (c *Client) FetchAll(ctx context.Context, keys []string) (map[string]Issue, error) {
	seen := make(map[string]bool, len(keys))
	var unique []string
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			unique = append(unique, k)
		}
	}

	if len(unique) == 0 {
		return map[string]Issue{}, nil
	}

	const batchSize = 50
	var batches [][]string
	for i := 0; i < len(unique); i += batchSize {
		end := min(i+batchSize, len(unique))
		batches = append(batches, unique[i:end])
	}

	sem := make(chan struct{}, 5)
	var mu sync.Mutex
	result := make(map[string]Issue)
	var firstErr error

	var wg sync.WaitGroup
	for _, batch := range batches {
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			issues, err := c.BulkFetch(ctx, batch)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			maps.Copy(result, issues)
		}(batch)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return result, nil
}

func extractTextFromADF(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}

	doc, ok := v.(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", v)
	}

	var b strings.Builder
	extractADFContent(&b, doc)
	return strings.TrimSpace(b.String())
}

func extractADFContent(b *strings.Builder, node map[string]any) {
	if text, ok := node["text"].(string); ok {
		fmt.Fprintf(b, "%s", text)
	}

	content, ok := node["content"].([]any)
	if !ok {
		return
	}

	for _, child := range content {
		childMap, ok := child.(map[string]any)
		if !ok {
			continue
		}
		nodeType, _ := childMap["type"].(string)
		extractADFContent(b, childMap)
		if nodeType == "paragraph" || nodeType == "heading" {
			fmt.Fprintf(b, "\n")
		}
	}
}
