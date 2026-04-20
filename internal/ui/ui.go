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

var hasColor = true

var symInfo, symSuccess, symError, symWarning string

var (
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // green
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))  // red
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // yellow
)

func init() {
	hasColor = term.IsTerminal(int(os.Stderr.Fd()))

	if supportsUnicode() {
		symInfo = "· "
		symSuccess = "✓ "
		symError = "✗ "
		symWarning = "! "
	} else {
		symInfo = "- "
		symSuccess = "+ "
		symError = "x "
		symWarning = "! "
	}
}

func supportsUnicode() bool {
	// JetBrains IDEs (GoLand, IntelliJ, etc.) use JediTerm which supports UTF-8
	if os.Getenv("TERMINAL_EMULATOR") != "" {
		return true
	}
	if runtime.GOOS == "windows" {
		// Windows Terminal
		if os.Getenv("WT_SESSION") != "" {
			return true
		}
		// VSCode, Hyper, etc.
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
	hasColor = v
}

var PauseFn = func(seconds float64) {
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}

func Info(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "%s%s\n", symInfo, msg)
}

func Error(w io.Writer, msg string) {
	sym := symError
	if hasColor {
		sym = styleError.Render(symError)
	}
	_, _ = fmt.Fprintf(w, "%s%s\n", sym, msg)
}

func Success(w io.Writer, msg string) {
	sym := symSuccess
	if hasColor {
		sym = styleSuccess.Render(symSuccess)
	}
	_, _ = fmt.Fprintf(w, "%s%s\n", sym, msg)
}

func Warning(w io.Writer, msg string) {
	sym := symWarning
	if hasColor {
		sym = styleWarning.Render(symWarning)
	}
	_, _ = fmt.Fprintf(w, "%s%s\n", sym, msg)
}

func Pause(seconds float64) {
	PauseFn(seconds)
}

func HintsBar(hints ...string) string {
	return "  " + strings.Join(hints, " · ") + "\n"
}
