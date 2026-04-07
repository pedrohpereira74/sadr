package dryrun

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	text := strings.Repeat("a", 400)
	tokens := EstimateTokens(text)
	if tokens != 100 {
		t.Errorf("expected 100 tokens, got %d", tokens)
	}
}

func TestEstimateTokensEmpty(t *testing.T) {
	if EstimateTokens("") != 0 {
		t.Error("expected 0 tokens for empty string")
	}
}

func TestFormatEstimate(t *testing.T) {
	result := FormatEstimate(23, 46080, 12000)
	if !strings.Contains(result, "23 records") {
		t.Errorf("expected record count in output: %s", result)
	}
	if !strings.Contains(result, "~45KB") {
		t.Errorf("expected payload size in output: %s", result)
	}
	if !strings.Contains(result, "~12000 tokens") {
		t.Errorf("expected token count in output: %s", result)
	}
}
