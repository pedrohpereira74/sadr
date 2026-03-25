package ui

import (
	"flag"
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

func Pause(seconds float64) {
	if os.Getenv("SADR_TEST") == "1" || flag.Lookup("test.v") != nil {
		return
	}
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}
