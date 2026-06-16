package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

type providerSpec struct {
	defaultModel string
	keyEnvs      []string
	url          func(model string) string
	headers      func(apiKey string) map[string]string
	body         func(model, prompt string) any
	parse        func(raw []byte) (string, error)
}

func geminiSpec() providerSpec {
	return providerSpec{
		defaultModel: DefaultModel,
		keyEnvs:      []string{"GEMINI_API_KEY"},
		url: func(model string) string {
			return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
		},
		headers: func(apiKey string) map[string]string {
			return map[string]string{"Content-Type": "application/json", "x-goog-api-key": apiKey}
		},
		body: func(_, prompt string) any {
			return map[string]any{
				"contents": []map[string]any{
					{"parts": []map[string]string{{"text": prompt}}},
				},
			}
		},
		parse: func(raw []byte) (string, error) {
			var r struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}
			if err := json.Unmarshal(raw, &r); err != nil {
				return "", fmt.Errorf("failed to parse API response: %v", err)
			}
			if len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
				return "", fmt.Errorf("empty response from API")
			}
			return r.Candidates[0].Content.Parts[0].Text, nil
		},
	}
}

func claudeSpec() providerSpec {
	return providerSpec{
		defaultModel: "claude-sonnet-4-6",
		keyEnvs:      []string{"ANTHROPIC_API_KEY", "CLAUDE_API_KEY"},
		url:          func(string) string { return "https://api.anthropic.com/v1/messages" },
		headers: func(apiKey string) map[string]string {
			return map[string]string{
				"Content-Type":      "application/json",
				"x-api-key":         apiKey,
				"anthropic-version": "2023-06-01",
			}
		},
		body: func(model, prompt string) any {
			return map[string]any{
				"model":      model,
				"max_tokens": 8192,
				"messages":   []map[string]string{{"role": "user", "content": prompt}},
			}
		},
		parse: func(raw []byte) (string, error) {
			var r struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			}
			if err := json.Unmarshal(raw, &r); err != nil {
				return "", fmt.Errorf("failed to parse API response: %v", err)
			}
			for _, c := range r.Content {
				if c.Type == "text" && c.Text != "" {
					return c.Text, nil
				}
			}
			return "", fmt.Errorf("empty response from API")
		},
	}
}

func openAICompatSpec(base, defaultModel string, keyEnvs []string) providerSpec {
	return providerSpec{
		defaultModel: defaultModel,
		keyEnvs:      keyEnvs,
		url:          func(string) string { return base + "/chat/completions" },
		headers: func(apiKey string) map[string]string {
			return map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + apiKey}
		},
		body: func(model, prompt string) any {
			return map[string]any{
				"model":    model,
				"messages": []map[string]string{{"role": "user", "content": prompt}},
			}
		},
		parse: func(raw []byte) (string, error) {
			var r struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			}
			if err := json.Unmarshal(raw, &r); err != nil {
				return "", fmt.Errorf("failed to parse API response: %v", err)
			}
			if len(r.Choices) == 0 {
				return "", fmt.Errorf("empty response from API")
			}
			return r.Choices[0].Message.Content, nil
		},
	}
}

func providerByName(name string) (providerSpec, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "gemini", "google":
		return geminiSpec(), nil
	case "claude", "anthropic":
		return claudeSpec(), nil
	case "openai", "gpt":
		return openAICompatSpec("https://api.openai.com/v1", "gpt-4o", []string{"OPENAI_API_KEY"}), nil
	case "deepseek":
		return openAICompatSpec("https://api.deepseek.com/v1", "deepseek-chat", []string{"DEEPSEEK_API_KEY"}), nil
	default:
		return providerSpec{}, fmt.Errorf("unknown AI provider %q (supported: gemini, claude, openai, deepseek)", name)
	}
}
