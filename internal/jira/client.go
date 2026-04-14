package jira

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type ClientConfig struct {
	BaseURL           string
	Username          string
	Password          string
	PasswordEnv       string
	Token             string
	TokenEnv          string
	ConsumerKey       string
	PrivateKeyPath    string
	AccessToken       string
	AccessTokenSecret string
}

type Client struct {
	BaseURL           string
	HTTP              *http.Client
	Username          string
	Password          string
	Token             string
	ConsumerKey       string
	PrivateKey        *rsa.PrivateKey
	AccessToken       string
	AccessTokenSecret string
}

type Issue struct {
	Key         string
	Summary     string
	Status      string
	Assignee    string
	Description string
}

type searchRequest struct {
	JQL        string   `json:"jql"`
	Fields     []string `json:"fields"`
	MaxResults int      `json:"maxResults"`
}

type searchResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string         `json:"summary"`
			Status      *statusField   `json:"status"`
			Assignee    *assigneeField `json:"assignee"`
			Description any            `json:"description"`
		} `json:"fields"`
	} `json:"issues"`
}

type statusField struct {
	Name string `json:"name"`
}

type assigneeField struct {
	DisplayName string `json:"displayName"`
}

func (c *Client) applyAuth(req *http.Request) error {
	if c.ConsumerKey != "" && c.PrivateKey != nil && c.AccessToken != "" {
		return c.applyOAuth(req)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
		return nil
	}
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
		return nil
	}
	return nil
}

func (c *Client) applyOAuth(req *http.Request) error {
	nonceBytes := make([]byte, 16)
	_, _ = rand.Read(nonceBytes)
	nonce := fmt.Sprintf("%x", nonceBytes)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	params := map[string]string{
		"oauth_consumer_key":     c.ConsumerKey,
		"oauth_token":            c.AccessToken,
		"oauth_signature_method": "RSA-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_nonce":            nonce,
		"oauth_version":          "1.0",
	}

	baseURL := req.URL.Scheme + "://" + req.URL.Host + req.URL.Path

	var paramParts []string
	for k, v := range params {
		paramParts = append(paramParts, percentEncode(k)+"="+percentEncode(v))
	}
	sort.Strings(paramParts)
	normalizedParams := strings.Join(paramParts, "&")

	baseString := strings.ToUpper(req.Method) + "&" +
		percentEncode(baseURL) + "&" +
		percentEncode(normalizedParams)

	h := sha1.New()
	h.Write([]byte(baseString))
	digest := h.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, c.PrivateKey, crypto.SHA1, digest)
	if err != nil {
		return fmt.Errorf("oauth signing failed: %w", err)
	}

	params["oauth_signature"] = base64.StdEncoding.EncodeToString(sig)

	var headerParts []string
	for k, v := range params {
		headerParts = append(headerParts, percentEncode(k)+`="`+percentEncode(v)+`"`)
	}
	sort.Strings(headerParts)
	req.Header.Set("Authorization", "OAuth "+strings.Join(headerParts, ", "))
	return nil
}

func percentEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func (c *Client) searchIssues(ctx context.Context, keys []string) (map[string]Issue, error) {
	if len(keys) == 0 {
		return map[string]Issue{}, nil
	}

	quoted := make([]string, len(keys))
	for i, k := range keys {
		quoted[i] = `"` + k + `"`
	}
	jql := "key in (" + strings.Join(quoted, ",") + ")"

	body, err := json.Marshal(searchRequest{
		JQL:        jql,
		Fields:     []string{"summary", "status", "assignee", "description"},
		MaxResults: len(keys),
	})
	if err != nil {
		return nil, err
	}

	apiURL := strings.TrimRight(c.BaseURL, "/") + "/rest/api/2/search"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.applyAuth(req); err != nil {
		return nil, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("jira API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var sr searchResponse
	if err := json.Unmarshal(respBody, &sr); err != nil {
		return nil, fmt.Errorf("failed to parse jira response: %w", err)
	}

	result := make(map[string]Issue, len(sr.Issues))
	for _, issue := range sr.Issues {
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

			issues, err := c.searchIssues(ctx, batch)
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

func (c *Client) OAuthRequest(ctx context.Context, method, endpoint string, extraParams url.Values) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(extraParams.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := c.applyOAuth(req); err != nil {
		return "", err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("oauth error (%d): %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}

func NewClientFromConfig(cfg ClientConfig) *Client {
	password := cfg.Password
	if password == "" {
		password = os.Getenv(cfg.PasswordEnv)
	}
	token := cfg.Token
	if token == "" {
		token = os.Getenv(cfg.TokenEnv)
	}

	c := &Client{
		BaseURL:           cfg.BaseURL,
		HTTP:              &http.Client{Timeout: 10 * time.Second},
		Username:          cfg.Username,
		Password:          password,
		Token:             token,
		ConsumerKey:       cfg.ConsumerKey,
		AccessToken:       cfg.AccessToken,
		AccessTokenSecret: cfg.AccessTokenSecret,
	}

	if cfg.PrivateKeyPath != "" {
		if key, err := loadPrivateKey(cfg.PrivateKeyPath); err == nil {
			c.PrivateKey = key
		}
	}

	return c
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	expanded := expandHome(path)
	data, err := os.ReadFile(expanded)
	if err != nil {
		return nil, err
	}
	return ParseRSAPrivateKey(data)
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
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
		b.WriteString(text)
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
			b.WriteString("\n")
		}
	}
}
