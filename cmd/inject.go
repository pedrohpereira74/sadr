package cmd

import (
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
)
