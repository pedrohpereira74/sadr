package dryrun

import "fmt"

func EstimateTokens(text string) int {
	return len(text) / 4
}

func FormatEstimate(recordCount int, payloadSize int, tokenEstimate int) string {
	return fmt.Sprintf(
		"dry run: %d records, ~%dKB payload, ~%d tokens estimated.",
		recordCount, payloadSize/1024, tokenEstimate,
	)
}
