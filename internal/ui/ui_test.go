package ui

import (
	"bytes"
	"testing"
)

func TestInfoWritesToWriter(t *testing.T) {
	SetTTY(true)
	var buf bytes.Buffer
	Info(&buf, "sadr-001 saved — Congrats!")

	output := buf.String()
	if output != ":(  sadr-001 saved — Congrats!\n" {
		t.Errorf("expected brand voice output, got '%s'", output)
	}
}

func TestErrorWritesToWriter(t *testing.T) {
	var buf bytes.Buffer
	Error(&buf, "Something went wrong")

	output := buf.String()
	if output != ":(  Something went wrong\n" {
		t.Errorf("expected error output, got '%s'", output)
	}
}

func TestInfoSuppressedWhenNotTTY(t *testing.T) {
	var buf bytes.Buffer
	SetTTY(false)
	defer SetTTY(true)

	Info(&buf, "should not appear")

	if buf.Len() != 0 {
		t.Errorf("expected empty output when not TTY, got '%s'", buf.String())
	}
}

func TestErrorAlwaysShows(t *testing.T) {
	var buf bytes.Buffer
	SetTTY(false)
	defer SetTTY(true)

	Error(&buf, "always visible")

	if buf.Len() == 0 {
		t.Error("expected error to show even when not TTY")
	}
}
