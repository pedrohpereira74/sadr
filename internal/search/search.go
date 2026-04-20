package search

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/sahilm/fuzzy"
)

var highlightStyle = lipgloss.NewStyle().Bold(true).Underline(true)

func Matches(r model.Record, query string, deep bool) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(r.Title), q) {
		return true
	}
	if MatchesTags(r, query) {
		return true
	}
	if deep && MatchesDeep(r, query) {
		return true
	}
	return false
}

func MatchesTags(r model.Record, query string) bool {
	return strings.Contains(strings.ToLower(strings.Join(r.Tags, ",")), strings.ToLower(query))
}

func MatchesDeep(r model.Record, query string) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(r.Snippet), q) {
		return true
	}
	for _, value := range r.Fields {
		if strings.Contains(strings.ToLower(value), q) {
			return true
		}
	}
	return false
}

func DeepContext(r model.Record, query string) string {
	if ctx := fuzzyContext(r.Snippet, query); ctx != "" {
		return ctx
	}
	for _, key := range r.FieldOrder {
		if ctx := fuzzyContext(r.Fields[key], query); ctx != "" {
			return ctx
		}
	}
	return ""
}

func fuzzyContext(text, query string) string {
	if text == "" {
		return ""
	}
	words := strings.Fields(text)
	text = strings.Join(words, " ")
	if idx := strings.Index(strings.ToLower(text), strings.ToLower(query)); idx >= 0 {
		return extractContext(text, idx, len(query))
	}
	results := fuzzy.Find(query, words)
	if len(results) == 0 {
		return ""
	}
	bestWord := results[0].Str
	idx := strings.Index(strings.ToLower(text), strings.ToLower(bestWord))
	if idx < 0 {
		return ""
	}
	return extractContext(text, idx, len(bestWord))
}

func extractContext(text string, idx, matchLen int) string {
	const maxLen = 30
	const dots = 3
	window := maxLen - dots*2
	half := 0
	if matchLen < window {
		half = (window - matchLen) / 2
	}
	start := max(idx-half, 0)
	end := min(start+window, len(text))
	if end-start < window {
		start = max(end-window, 0)
	}
	idx = max(idx, start)
	idx = min(idx, end)
	matchEnd := min(idx+matchLen, end)
	before := text[start:idx]
	match := highlightStyle.Render(text[idx:matchEnd])
	after := text[matchEnd:end]
	result := before + match + after
	if start > 0 {
		result = "..." + result
	}
	if end < len(text) {
		result += "..."
	}
	return result
}
