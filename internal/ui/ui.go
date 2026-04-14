package ui

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var isTTY = true

var symInfo, symSuccess, symError, symWarning string

var (
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // green
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))  // red
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // yellow
)

func init() {
	isTTY = term.IsTerminal(int(os.Stdout.Fd()))

	if supportsUnicode() {
		symInfo = "· "
		symSuccess = "✓ "
		symError = "✗ "
		symWarning = "! "
	} else {
		symInfo = "  "
		symSuccess = "+ "
		symError = "x "
		symWarning = "! "
	}
}

func supportsUnicode() bool {
	if runtime.GOOS == "windows" {
		if os.Getenv("WT_SESSION") != "" {
			return true
		}
		if os.Getenv("TERM_PROGRAM") != "" {
			return true
		}
		t := os.Getenv("TERM")
		return t != "" && t != "dumb"
	}
	for _, env := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		val := strings.ToUpper(os.Getenv(env))
		if strings.Contains(val, "UTF-8") || strings.Contains(val, "UTF8") {
			return true
		}
	}
	return os.Getenv("TERM_PROGRAM") != ""
}

func setTTY(v bool) {
	isTTY = v
}

var PauseFn = func(seconds float64) {
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}

func Info(w io.Writer, msg string) {
	if !isTTY {
		return
	}
	_, _ = fmt.Fprintf(w, "%s%s\n", symInfo, msg)
}

func Error(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "%s%s\n", styleError.Render(symError), msg)
}

func Success(w io.Writer, msg string) {
	if !isTTY {
		return
	}
	_, _ = fmt.Fprintf(w, "%s%s\n", styleSuccess.Render(symSuccess), msg)
}

func Warning(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "%s%s\n", styleWarning.Render(symWarning), msg)
}

func Pause(seconds float64) {
	PauseFn(seconds)
}
