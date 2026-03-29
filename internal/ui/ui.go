package ui

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

var isTTY = true

func init() {
	isTTY = term.IsTerminal(int(os.Stdout.Fd()))
}

func SetTTY(v bool) {
	isTTY = v
}

var PauseFn = func(seconds float64) {
	time.Sleep(time.Duration(seconds * float64(time.Second)))
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
	_, _ = fmt.Fprintf(w, ":(  %s\n", msg)
}

func Pause(seconds float64) {
	PauseFn(seconds)
}
