package ui

import (
	"bytes"
	"testing"
)

func TestInfoWritesToWriter(t *testing.T) {
	var buf bytes.Buffer
	Info(&buf, "sadr-001 saved — Congrats!")

	output := buf.String()
	if output != symInfo+"sadr-001 saved — Congrats!\n" {
		t.Errorf("expected info output, got '%s'", output)
	}
}

func TestInfoAlwaysShows(t *testing.T) {
	var buf bytes.Buffer
	setTTY(false)
	defer setTTY(true)

	Info(&buf, "always visible")

	if buf.Len() == 0 {
		t.Error("expected info to show even when not TTY")
	}
}

func TestErrorWritesToWriter(t *testing.T) {
	setTTY(false)
	defer setTTY(true)

	var buf bytes.Buffer
	Error(&buf, "Something went wrong")

	output := buf.String()
	if output != symError+"Something went wrong\n" {
		t.Errorf("expected error output, got '%s'", output)
	}
}

func TestErrorAlwaysShows(t *testing.T) {
	var buf bytes.Buffer
	setTTY(false)
	defer setTTY(true)

	Error(&buf, "always visible")

	if buf.Len() == 0 {
		t.Error("expected error to show even when not TTY")
	}
}

func TestSuccessAlwaysShows(t *testing.T) {
	var buf bytes.Buffer
	setTTY(false)
	defer setTTY(true)

	Success(&buf, "record saved")

	if buf.Len() == 0 {
		t.Error("expected success to show even when not TTY")
	}
}

func TestWarningAlwaysShows(t *testing.T) {
	var buf bytes.Buffer
	setTTY(false)
	defer setTTY(true)

	Warning(&buf, "heads up")

	if buf.Len() == 0 {
		t.Error("expected warning to show even when not TTY")
	}
}

func TestPauseCallsInjectedFn(t *testing.T) {
	called := false
	old := PauseFn
	PauseFn = func(_ float64) { called = true }
	defer func() { PauseFn = old }()

	Pause(1.0)

	if !called {
		t.Error("expected PauseFn to be called by Pause")
	}
}

func TestPausePassesSeconds(t *testing.T) {
	var got float64
	old := PauseFn
	PauseFn = func(s float64) { got = s }
	defer func() { PauseFn = old }()

	Pause(2.5)

	if got != 2.5 {
		t.Errorf("expected 2.5 seconds, got %f", got)
	}
}
