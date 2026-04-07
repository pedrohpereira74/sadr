package cmd

import (
	"context"
	"time"

	"github.com/pedrohpereira74/sadr/internal/ai"
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

	generateTextFn func(ctx context.Context, prompt, apiKey, model string, timeout time.Duration) (string, error) = ai.GenerateText
	confirmPromptFn func(message string) bool = confirmPromptImpl
)
