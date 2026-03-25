package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/storage"
)

func setupNewTest(t *testing.T) string {
	t.Helper()
	newTitle = ""
	newQuick = false
	newGlobal = false
	newClipboard = false
	newFile = ""
	newDiff = false
	newSmart = false

	dir, _ := os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	sadrDir := filepath.Join(dir, ".sadr", "records")
	if err := os.MkdirAll(sadrDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	originalWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })
	return dir
}

func TestNewCreatesRecord(t *testing.T) {
	dir := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "--title", "Use retry with backoff"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	entries, err := os.ReadDir(recordsDir)
	if err != nil {
		t.Fatalf("failed to read records dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 record, got %d", len(entries))
	}
}

func TestNewAdrCreatesRecordWithTypeAdr(t *testing.T) {
	dir := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "adr", "--title", "Use Redis for cache"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	s := storage.NewStorage(recordsDir)
	records, _ := s.ListRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Type != "adr" {
		t.Errorf("expected type 'adr', got '%s'", records[0].Type)
	}
}

func TestNewSnippetCreatesRecordWithTypeSnippet(t *testing.T) {
	dir := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "snippet", "--title", "Retry helper"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	s := storage.NewStorage(recordsDir)
	records, _ := s.ListRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Type != "snippet" {
		t.Errorf("expected type 'snippet', got '%s'", records[0].Type)
	}
}

func TestNewQuickOnlyAsksQuickFields(t *testing.T) {
	dir := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "--quick", "--title", "Quick record"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	s := storage.NewStorage(recordsDir)
	records, _ := s.ListRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
}

func TestNewGlobalSavesToHome(t *testing.T) {
	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	globalRecords := filepath.Join(home, ".sadr", "records")
	if err := os.MkdirAll(globalRecords, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	rootCmd.SetArgs([]string{"new", "--global", "--title", "Personal snippet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(globalRecords)
	if len(entries) != 1 {
		t.Errorf("expected 1 record in global, got %d", len(entries))
	}
}

func TestNewReadsConfigForWizard(t *testing.T) {
	dir := setupNewTest(t)

	configContent := `
fields:
  - name: title
    type: text
    required: true
  - name: tags
    type: multiselect
    required: true
    options: [api, security, database]
`
	configPath := filepath.Join(dir, ".sadr", "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	rootCmd.SetArgs([]string{"new", "--title", "Test record"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 record, got %d", len(entries))
	}
}

func TestNewFromFile(t *testing.T) {
	dir := setupNewTest(t)

	snippetFile := filepath.Join(dir, "snippet.go")
	if err := os.WriteFile(snippetFile, []byte("client := retryablehttp.NewClient()"), 0644); err != nil {
		t.Fatalf("failed to write snippet file: %v", err)
	}

	rootCmd.SetArgs([]string{"new", "--title", "Retry client", "--file", snippetFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	s := storage.NewStorage(recordsDir)
	records, _ := s.ListRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Snippet != "client := retryablehttp.NewClient()" {
		t.Errorf("expected snippet from file, got '%s'", records[0].Snippet)
	}
}

func TestNewFromDiff(t *testing.T) {
	dir := setupNewTest(t)

	_ = exec.Command("git", "init").Run()
	_ = os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main"), 0644)
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "initial").Run()
	_ = os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main\n\nfunc hello() {}"), 0644)

	rootCmd.SetArgs([]string{"new", "--diff", "--title", "Added hello function"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", "records")
	s := storage.NewStorage(recordsDir)
	records, _ := s.ListRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Snippet == "" {
		t.Error("expected snippet from git diff, got empty")
	}
}

func TestNewFromDiffEmpty(t *testing.T) {
	dir := setupNewTest(t)

	_ = exec.Command("git", "init").Run()
	_ = os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main"), 0644)
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "initial").Run()

	rootCmd.SetArgs([]string{"new", "--diff", "--title", "No changes"})
	_ = rootCmd.Execute()

	recordsDir := filepath.Join(dir, ".sadr", "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 0 {
		t.Errorf("expected 0 records when diff is empty, got %d", len(entries))
	}
}

func TestNewSmartWithoutSnippetAborts(t *testing.T) {
	dir := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "--smart"})
	t.Setenv("EDITOR", "sort")

	_ = rootCmd.Execute()

	recordsDir := filepath.Join(dir, ".sadr", "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 0 {
		t.Errorf("expected 0 records when --smart is aborted, got %d", len(entries))
	}
}

func TestNewSmartWithoutAPIKeyStillCreatesRecord(t *testing.T) {
	dir := setupNewTest(t)
	t.Setenv("GEMINI_API_KEY", "")

	snippetFile := filepath.Join(dir, "snippet.go")
	_ = os.WriteFile(snippetFile, []byte("client := retryablehttp.NewClient()"), 0644)

	rootCmd.SetArgs([]string{"new", "--smart", "--file", snippetFile, "--title", "Test"})
	_ = rootCmd.Execute()

	recordsDir := filepath.Join(dir, ".sadr", "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 record (fallback to manual), got %d", len(entries))
	}
}
