package ask

import (
	"fmt"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/enricher"
)

type Persona struct {
	Name        string
	Instruction string
}

func BuildAskPrompt(persona Persona, question string, contexts []enricher.RecordContext, language string, withSnippet bool) string {
	if language == "" {
		language = "English"
	}

	var b strings.Builder

	fmt.Fprintf(&b, `You are a %s. %s

A developer asked you a direct question. Your job is to answer it — not to summarize all available records.

Rules:
- Match the depth of your answer to the complexity of the question. A simple inventory question ("what do we have?", "list the records") deserves a concise list — not an architectural analysis.
- Start with a direct answer to the question in 1-2 sentences.
- Then back it up with specific evidence from the records below, only if it adds value.
- Select only what is relevant to the question. Skip unrelated records entirely.
- Do NOT introduce yourself or add any personal opening.
- Do NOT speculate beyond what is present in the records.
- Do NOT volunteer analysis that was not asked for.
- You MUST write entirely in %s.

QUESTION: %s

RECORDS AND CONTEXT:
`, persona.Name, persona.Instruction, language, question)

	for i, ctx := range contexts {
		fmt.Fprintf(&b, "\n--- RECORD %d: %s (type: %s)\n", i+1, ctx.RecordTitle, ctx.RecordType)

		if ctx.RecordTags != "" {
			fmt.Fprintf(&b, "Tags: %s\n", ctx.RecordTags)
		}

		for key, val := range ctx.RecordFields {
			if key == "tags" || val == "" {
				continue
			}
			fmt.Fprintf(&b, "%s: %s\n", key, val)
		}


		if withSnippet && ctx.RecordSnippet != "" {
			fmt.Fprintf(&b, "Snippet:\n```\n%s\n```\n", enricher.ZipSnippet(ctx.RecordSnippet))
		}

		for _, sf := range ctx.SourceFiles {
			if sf.SourceCode != "" {
				fmt.Fprintf(&b, "<file_ref path=%q>\n%s\n</file_ref>\n", sf.SourcePath, enricher.ZipSourceCode(sf.SourceCode))
			}
			if sf.TestCode != "" {
				fmt.Fprintf(&b, "<file_ref path=%q>\n%s\n</file_ref>\n", sf.TestPath, enricher.ZipSourceCode(sf.TestCode))
			}
		}

		if ctx.JiraIssue != nil {
			fmt.Fprintf(&b, "Jira: %s — %s (%s)\n", ctx.JiraIssue.Key, ctx.JiraIssue.Summary, ctx.JiraIssue.Status)
		}
	}

	return b.String()
}
