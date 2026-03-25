package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const DefaultModel = "gemini-3-flash-preview"

func BuildPrompt(snippet string, fields []string, language string) string {
	if language == "" {
		language = "English"
	}

	return fmt.Sprintf(`You are an expert Software Architect and Code Documentation Assistant.
Your task is to analyze the provided code snippet and extract or deduce the requested metadata.

CRITICAL RULES - FAILURE TO COMPLY WILL BREAK THE SYSTEM:
1. STRICT JSON ONLY: You must output ONLY a valid, raw JSON object.
2. NO MARKDOWN: ABSOLUTELY NO backticks, NO 'json' codeblock tags, and NO conversational text before or after the JSON.
3. EXACT KEYS: Your JSON MUST contain EXACTLY the following keys:
[%s]
DO NOT translate, rename, add, or omit any keys. The keys must remain exactly as listed above.
4. CONTENT LANGUAGE: The VALUES inside the JSON must be written in: %s.

CODE SNIPPET TO ANALYZE:
---
%s
---`, strings.Join(fields, ", "), language, snippet)
}

func ParseResponse(response string) (map[string]string, error) {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(response), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	result := map[string]string{}
	for key, val := range raw {
		cleanKey := strings.ToLower(strings.TrimSpace(key))

		switch v := val.(type) {
		case string:
			result[cleanKey] = v
		case []interface{}:
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

func Suggest(snippet string, fields []string, language string, apiKey string) (map[string]string, error) {
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY") // legacy fallback
	}
	if apiKey == "" {
		return nil, fmt.Errorf("AI API key not set in global config or environment")
	}

	prompt := BuildPrompt(snippet, fields, language)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", DefaultModel, apiKey)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("API request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
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
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text
	return ParseResponse(text)
}
