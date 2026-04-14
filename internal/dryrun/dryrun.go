package dryrun

import (
	"fmt"

	"github.com/pedrohpereira74/sadr/internal/compress"
	"github.com/pedrohpereira74/sadr/internal/enricher"
)

func estimateTokens(text string) int {
	return len(text) / 4
}

func FormatEstimate(recordCount int, payloadSize int, tokenEstimate int) string {
	return fmt.Sprintf(
		"dry run: %d records, ~%dKB payload, ~%d tokens estimated.",
		recordCount, payloadSize/1024, tokenEstimate,
	)
}

// ContextsPayloadSize returns the estimated byte size of all record contexts,
// including source files when withSnippet is true.
func ContextsPayloadSize(contexts []enricher.RecordContext, withSnippet bool) int {
	var size int
	for _, ctx := range contexts {
		size += len(ctx.RecordTitle)
		if withSnippet && ctx.RecordSnippet != "" {
			size += len(compress.ZipSnippet(ctx.RecordSnippet))
		}
		for _, v := range ctx.RecordFields {
			size += len(v)
		}
		for _, sf := range ctx.SourceFiles {
			if sf.SourceCode != "" {
				size += len(compress.ZipSourceCode(sf.SourceCode))
			}
			if sf.TestCode != "" {
				size += len(compress.ZipSourceCode(sf.TestCode))
			}
		}
	}
	return size
}
