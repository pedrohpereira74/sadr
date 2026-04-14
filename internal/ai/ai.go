package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	DefaultModel = "gemini-3-flash-preview"

	basePersona = "You are an expert Software Architect and Code Documentation Assistant."

	staffPersona = "You are a Staff-level Software Engineer and Tech Lead performing a deep code review."

	depthInstructions = `
DEPTH INSTRUCTIONS - APPLY ONLY TO PERTINENT FIELDS:
8. For "text" fields: Write DETAILED, thorough explanations. Minimum 2-3 sentences. Analyze trade-offs, hidden coupling, performance implications, and architectural debt. Be precise and opinionated. Justify your reasoning with concrete technical arguments.
9. For "list" fields: Return items separated by commas with NO bullet points. You MUST use descriptive full phrases, not single words or identifiers (e.g. "Implementing a dedicated Auth microservice, Using a standard OAuth flow").`

	promptTemplate = `%s
Your task is to analyze the provided code snippet and extract or deduce the requested metadata.

CRITICAL RULES - FAILURE TO COMPLY WILL BREAK THE SYSTEM:
1. STRICT JSON ONLY: You must output ONLY a valid, raw JSON object.
2. NO MARKDOWN: ABSOLUTELY NO backticks, NO 'json' codeblock tags, and NO conversational text before or after the JSON.
3. EXACT KEYS: Your JSON MUST contain EXACTLY the following keys:
[%s]
DO NOT translate, rename, add, or omit any keys. The keys must remain exactly as listed above.
4. CONTENT LANGUAGE: The VALUES inside the JSON must be written in: %s.
5. VOICE: Write in third person, impersonal, technical report style. NEVER use first person ("I", "we", "my", "our"). Describe what the code does, not what you did or observed.
6. EMPTY FIELDS: If you cannot provide a meaningful, specific answer for a field based on the snippet (e.g. no real alternatives exist, the information is not inferable), return "none identified" translated to the content language defined in rule 4. DO NOT fabricate or pad answers just to fill a field.
7. TITLE LENGTH: If a "title" key is present, it MUST be a short, concise phrase of at most 10 words. Do NOT write full sentences or descriptions as titles.
8. FIELD TYPES: Some keys above may include a type hint in parentheses, e.g. "consequences (text)". Use ONLY the key name (without the hint) in the JSON output. Respect the type:
   - "(text)": Write complete, flowing prose. Use full sentences separated by periods. Do NOT use comma-separated lists.
   - "(list)": Return items separated by commas with NO bullet points.
   If no type hint is given, default to text behavior.
%s
%sCODE SNIPPET TO ANALYZE:
---
%s
---`
)

func BuildPrompt(snippet string, fields []string, language string, depth bool, jiraContext string) string {
	if language == "" {
		language = "English"
	}

	persona := basePersona
	depthBlock := ""

	if depth {
		persona = staffPersona
		depthBlock = depthInstructions
	}

	jiraSection := ""
	if jiraContext != "" {
		jiraSection = "JIRA ISSUE CONTEXT:\n---\n" + jiraContext + "\n---\n\n"
	}

	return fmt.Sprintf(promptTemplate, persona, strings.Join(fields, ", "), language, depthBlock, jiraSection, snippet)
}

func ParseResponse(response string) (map[string]string, error) {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var raw map[string]any
	if err := json.Unmarshal([]byte(response), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	result := map[string]string{}
	for key, val := range raw {
		cleanKey := strings.ToLower(strings.TrimSpace(key))

		switch v := val.(type) {
		case string:
			result[cleanKey] = v
		case []any:
			var parts []string
			for _, item := range v {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
			result[cleanKey] = strings.Join(parts, ", ")
		default:
			result[cleanKey] = fmt.Sprintf("%v", v)
		}
	}

	return result, nil
}

var httpClient = &http.Client{Timeout: 90 * time.Second}

func resolveAPIKey(apiKey string) (string, error) {
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("AI API key not set in global config or environment")
	}
	return apiKey, nil
}

func GenerateText(ctx context.Context, prompt string, apiKey string, model string, timeout time.Duration) (string, error) {
	apiKey, err := resolveAPIKey(apiKey)
	if err != nil {
		return "", err
	}

	if model == "" {
		model = DefaultModel
	}

	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)

	client := httpClient
	if timeout > 0 {
		client = &http.Client{
			Timeout:   timeout,
			Transport: httpClient.Transport,
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %v", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func Suggest(ctx context.Context, snippet string, fields []string, language string, apiKey string, model string, depth bool, jiraContext string) (map[string]string, error) {
	prompt := BuildPrompt(snippet, fields, language, depth, jiraContext)
	text, err := GenerateText(ctx, prompt, apiKey, model, 0)
	if err != nil {
		return nil, err
	}
	return ParseResponse(text)
}
