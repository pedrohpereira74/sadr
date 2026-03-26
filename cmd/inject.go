package cmd

import (
	"github.com/pedrohpereira74/sadr/internal/wizard"
)

// Injection points for interactive functions to allow hermetic testing.
// In production, these point to the real implementations.
// In tests, these can be swapped for mocks.
var (
	editorRunner = openEditorImpl
	wizardRunner = wizard.Run
	presetSelector = selectPresetImpl
	fallbackPrompter = promptGlobalFallbackImpl
	snippetCapturer = captureSnippetFromEditorImpl
)
