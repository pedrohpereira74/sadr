package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedrohpereira74/sadr/internal/ask"
	"github.com/pedrohpereira74/sadr/internal/config"
	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func makeEntry(title string, tags string, status string, createdAt time.Time) storage.RecordEntry {
	r, _ := model.NewRecordWithOptions(title, "full")
	r.Fields["tags"] = tags
	r.Fields["status"] = status
	r.CreatedAt = createdAt
	return storage.RecordEntry{Record: r, FileID: 1}
}

func TestFilterRecordEntriesByTag(t *testing.T) {
	entries := []storage.RecordEntry{
		makeEntry("API design", "api,architecture", "active", time.Now()),
		makeEntry("DB indexing", "database,performance", "active", time.Now()),
	}
	opts := &askOptions{tags: "api"}
	result := filterRecordEntries(entries, opts, config.AskConfig{})
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
	if result[0].Record.Title != "API design" {
		t.Errorf("unexpected title: %s", result[0].Record.Title)
	}
}

func TestFilterRecordEntriesByField(t *testing.T) {
	r1, _ := model.NewRecordWithOptions("Service A", "full")
	r1.Fields["env"] = "prod"
	r2, _ := model.NewRecordWithOptions("Service B", "full")
	r2.Fields["env"] = "staging"
	entries := []storage.RecordEntry{
		{Record: r1, FileID: 1},
		{Record: r2, FileID: 2},
	}
	opts := &askOptions{field: "env=prod"}
	result := filterRecordEntries(entries, opts, config.AskConfig{})
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
	if result[0].Record.Title != "Service A" {
		t.Errorf("unexpected title: %s", result[0].Record.Title)
	}
}

func TestFilterRecordEntriesExcludesInactiveStatuses(t *testing.T) {
	entries := []storage.RecordEntry{
		makeEntry("Active", "", "active", time.Now()),
		makeEntry("Proposed", "", "proposed", time.Now()),
		makeEntry("Deprecated", "", "deprecated", time.Now()),
		makeEntry("Superseded", "", "superseded", time.Now()),
	}
	opts := &askOptions{}
	result := filterRecordEntries(entries, opts, config.AskConfig{})
	if len(result) != 1 {
		t.Errorf("expected 1 result (only active), got %d", len(result))
	}
	if result[0].Record.Title != "Active" {
		t.Errorf("unexpected title: %s", result[0].Record.Title)
	}
}

func TestFilterRecordEntriesRangeCutoff(t *testing.T) {
	old := makeEntry("Old record", "", "active", time.Now().AddDate(-2, 0, 0))
	recent := makeEntry("Recent record", "", "active", time.Now())
	entries := []storage.RecordEntry{old, recent}

	opts := &askOptions{}
	result := filterRecordEntries(entries, opts, config.AskConfig{Range: "6m"})
	if len(result) != 1 {
		t.Errorf("expected 1 result after cutoff, got %d", len(result))
	}
	if result[0].Record.Title != "Recent record" {
		t.Errorf("unexpected title: %s", result[0].Record.Title)
	}
}

func TestFilterRecordEntriesLimit(t *testing.T) {
	var entries []storage.RecordEntry
	for i := range 10 {
		e := makeEntry("Record", "", "active", time.Now().Add(time.Duration(i)*time.Second))
		e.FileID = i + 1
		entries = append(entries, e)
	}
	opts := &askOptions{}
	result := filterRecordEntries(entries, opts, config.AskConfig{Limit: 3})
	if len(result) != 3 {
		t.Errorf("expected 3 results with limit, got %d", len(result))
	}
}

func TestFilterRecordEntriesCombined(t *testing.T) {
	entries := []storage.RecordEntry{
		makeEntry("Match", "api", "active", time.Now()),
		makeEntry("Wrong tag", "database", "active", time.Now()),
		makeEntry("Deprecated match", "api", "deprecated", time.Now()),
	}
	opts := &askOptions{tags: "api"}
	result := filterRecordEntries(entries, opts, config.AskConfig{})
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestWriteAnswerFileHappyPath(t *testing.T) {
	dir := t.TempDir()
	answersDir := filepath.Join(dir, "answers")

	persona := ask.Persona{Name: "Tech Lead"}
	path, err := writeAnswerFile(answersDir, persona, "how to scale?", "use caching")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		t.Errorf("expected file to exist at %s", path)
	}

	base := filepath.Base(path)
	if !strings.HasPrefix(base, "sadr-answer-") {
		t.Errorf("unexpected filename pattern: %s", base)
	}

	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "Tech Lead") {
		t.Error("expected persona name in file content")
	}
	if !strings.Contains(string(content), "how to scale?") {
		t.Error("expected question in file content")
	}
}

func TestWriteAnswerFileCreatesDir(t *testing.T) {
	dir := t.TempDir()
	answersDir := filepath.Join(dir, "new-answers-dir")

	persona := ask.Persona{Name: "DBA"}
	_, err := writeAnswerFile(answersDir, persona, "question", "answer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(answersDir); os.IsNotExist(statErr) {
		t.Error("expected answers directory to be created")
	}
}

func TestWriteAnswerFileErrorOnBadPath(t *testing.T) {
	persona := ask.Persona{Name: "QA"}
	_, err := writeAnswerFile("/nonexistent/deeply/nested/path", persona, "q", "a")
	if err == nil {
		t.Error("expected error for unwritable path, got nil")
	}
}
