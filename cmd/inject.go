package cmd

import (
	"context"
	"time"

	"github.com/pedrohpereira74/sadr/internal/ai"
	jiraclient "github.com/pedrohpereira74/sadr/internal/jira"
	"github.com/pedrohpereira74/sadr/internal/wizard"
)

var (
	editorRunner     func(string, string) error = openEditorImpl
	wizardRunner                                = wizard.Run
	presetSelector                              = selectPresetImpl
	fallbackPrompter                            = promptGlobalFallbackImpl
	snippetCapturer                             = captureSnippetFromEditorImpl
	clipboardReader                             = readClipboardImpl
	confirmOverwrite                            = confirmOverwriteImpl
	gitTopLevelFn                               = gitTopLevelImpl
	gitDiffFn                                   = gitDiffImpl
	gitCommitFn                                 = gitCommitImpl

	generateTextFn  func(ctx context.Context, provider, prompt, apiKey, model string, timeout time.Duration) (string, error) = ai.GenerateText
	confirmPromptFn func(message string) bool                                                                      = confirmPromptImpl

	jiraFetcherFn func(ctx context.Context, client *jiraclient.Client, key string) (jiraclient.Issue, bool) = defaultJiraFetcher
)
