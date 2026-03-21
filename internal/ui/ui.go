package ui

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

var isTTY = true

func init() {
	isTTY = term.IsTerminal(int(os.Stdout.Fd()))
}

func SetTTY(v bool) {
	isTTY = v
}

func Info(w io.Writer, msg string) {
	if !isTTY {
		return
	}
	_, _ = fmt.Fprintf(w, ":(  %s\n", msg)
}

func Error(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, ":(  %s\n", msg)
}

func Success(w io.Writer, msg string) {
	if !isTTY {
		return
	}
	_, _ = fmt.Fprintf(w, ":(  %s\n", msg)
}

func Warning(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, ":(  Warning: %s\n", msg)
}
