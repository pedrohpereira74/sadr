package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/templates"
)

func setupNewTest(t *testing.T) (dir string, username string) {
	t.Helper()
	resetCmd(findSubCmd("new"))

	home, _ := os.MkdirTemp("", "sadr-test-home-*")
	t.Cleanup(func() { os.RemoveAll(home) })
	username = setupTestUser(t, home)

	dir, _ = os.MkdirTemp("", "sadr-test-*")
	t.Cleanup(func() { os.RemoveAll(dir) })
	configsDir := filepath.Join(dir, ".sadr", "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configsDir, "default-config.yaml"), []byte(templates.MinimalConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	originalWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })
	return dir, username
}

func TestNewCreatesRecord(t *testing.T) {
	dir, username := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "--title", "Use retry with backoff"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	entries, err := os.ReadDir(recordsDir)
	if err != nil {
		t.Fatalf("failed to read records dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 record, got %d", len(entries))
	}
}

func TestNewAdrCreatesRecordWithTypeAdr(t *testing.T) {
	dir, username := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "adr", "--title", "Use Redis for cache"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
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
	dir, username := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "snippet", "--title", "Retry helper"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	s := storage.NewStorage(recordsDir)
	records, _ := s.ListRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Type != "snippet" {
		t.Errorf("expected type 'snippet', got '%s'", records[0].Type)
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
	globalConfigs := filepath.Join(home, ".sadr", "configs")
	if err := os.MkdirAll(globalConfigs, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(globalConfigs, "default-config.yaml"), []byte(templates.MinimalConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
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
	dir, username := setupNewTest(t)

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
	configsDir := filepath.Join(dir, ".sadr", "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}
	configPath := filepath.Join(configsDir, "default-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	rootCmd.SetArgs([]string{"new", "--title", "Test record"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 record, got %d", len(entries))
	}
}

func TestNewFromFile(t *testing.T) {
	dir, username := setupNewTest(t)

	snippetFile := filepath.Join(dir, "snippet.go")
	if err := os.WriteFile(snippetFile, []byte("client := retryablehttp.NewClient()"), 0644); err != nil {
		t.Fatalf("failed to write snippet file: %v", err)
	}

	rootCmd.SetArgs([]string{"new", "--title", "Retry client", "--file", snippetFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
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
	dir, username := setupNewTest(t)

	_ = exec.Command("git", "init").Run()
	_ = exec.Command("git", "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "config", "user.name", "Test User").Run()
	_ = os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main"), 0644)
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "initial").Run()
	_ = os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main\n\nfunc hello() {}"), 0644)

	rootCmd.SetArgs([]string{"new", "--diff", "--title", "Added hello function"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
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
	dir, username := setupNewTest(t)

	_ = exec.Command("git", "init").Run()
	_ = exec.Command("git", "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "config", "user.name", "Test User").Run()
	_ = os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main"), 0644)
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "initial").Run()

	rootCmd.SetArgs([]string{"new", "--diff", "--title", "No changes"})
	_ = rootCmd.Execute()

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 0 {
		t.Errorf("expected 0 records when diff is empty, got %d", len(entries))
	}
}

func TestNewSmartWithoutSnippetAborts(t *testing.T) {
	dir, username := setupNewTest(t)

	rootCmd.SetArgs([]string{"new", "--smart"})
	t.Setenv("EDITOR", "sort")

	oldSnippetCapturer := snippetCapturer
	snippetCapturer = func() (string, error) { return "", nil }
	defer func() { snippetCapturer = oldSnippetCapturer }()

	_ = rootCmd.Execute()

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 0 {
		t.Errorf("expected 0 records when --smart is aborted, got %d", len(entries))
	}
}

func TestNewSmartWithoutAPIKeyStillCreatesRecord(t *testing.T) {
	dir, username := setupNewTest(t)
	t.Setenv("GEMINI_API_KEY", "")

	snippetFile := filepath.Join(dir, "snippet.go")
	_ = os.WriteFile(snippetFile, []byte("client := retryablehttp.NewClient()"), 0644)

	rootCmd.SetArgs([]string{"new", "--smart", "--file", snippetFile, "--title", "Test"})
	_ = rootCmd.Execute()

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	entries, _ := os.ReadDir(recordsDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 record (fallback to manual), got %d", len(entries))
	}
}

func TestExtractFilesFromDiff(t *testing.T) {
	diff := `diff --git a/src/handlers/auth.go b/src/handlers/auth.go
index 1234567..abcdefg 100644
--- a/src/handlers/auth.go
+++ b/src/handlers/auth.go
@@ -1,3 +1,5 @@
 package handlers`

	result := extractFilesFromDiff(diff)
	want := filepath.FromSlash("src/handlers/auth.go")
	if len(result) != 1 || result[0] != want {
		t.Errorf("expected [%q], got %v", want, result)
	}
}

func TestExtractFilesFromDiffEmpty(t *testing.T) {
	result := extractFilesFromDiff("")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestExtractFilesFromDiffMultipleFiles(t *testing.T) {
	diff := `diff --git a/file1.go b/file1.go
index 1234567..abcdefg 100644
diff --git a/file2.go b/file2.go
index 1234567..abcdefg 100644`

	result := extractFilesFromDiff(diff)
	if len(result) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result))
	}
	if result[0] != "file1.go" || result[1] != "file2.go" {
		t.Errorf("expected [file1.go file2.go], got %v", result)
	}
}

func TestNewFromFileSetsFileRef(t *testing.T) {
	dir, username := setupNewTest(t)

	snippetFile := filepath.Join(dir, "src", "main.go")
	if err := os.MkdirAll(filepath.Dir(snippetFile), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(snippetFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write snippet file: %v", err)
	}

	rootCmd.SetArgs([]string{"new", "--title", "Test file ref", "--file", snippetFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recordsDir := filepath.Join(dir, ".sadr", username, "records")
	s := storage.NewStorage(recordsDir)
	records, _ := s.ListRecords()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].FileRef == "" || records[0].FileRef == "N/A" {
		t.Errorf("expected file_ref to be set from --file, got '%s'", records[0].FileRef)
	}
}

func TestExtractFilesFromDiffDeduplicates(t *testing.T) {
	diff := `diff --git a/file1.go b/file1.go
index 1234567..abcdefg 100644
diff --git a/file1.go b/file1.go
index 1234567..abcdefg 100644`

	result := extractFilesFromDiff(diff)
	if len(result) != 1 {
		t.Fatalf("expected 1 file (deduplicated), got %d", len(result))
	}
}

func TestResolveProjectRootLocal(t *testing.T) {
	paths := discover.SadrPaths{
		Root:     filepath.Join(string(filepath.Separator)+"some", "project", ".sadr"),
		IsGlobal: false,
	}
	got := resolveProjectRoot(paths)
	want := filepath.Join(string(filepath.Separator)+"some", "project")
	if got != want {
		t.Errorf("local vault: got %q, want %q", got, want)
	}
}

func TestResolveProjectRootGlobalUsesGitRoot(t *testing.T) {
	oldGit := gitTopLevelFn
	defer func() { gitTopLevelFn = oldGit }()
	repoRoot := filepath.Join(string(filepath.Separator)+"home", "alice", "work", "myrepo")
	gitTopLevelFn = func() (string, error) { return repoRoot, nil }

	home, _ := os.UserHomeDir()
	paths := discover.SadrPaths{Root: filepath.Join(home, ".sadr"), IsGlobal: true}
	got := resolveProjectRoot(paths)
	if got != repoRoot {
		t.Errorf("global fallback should resolve to git repo root, got %q want %q", got, repoRoot)
	}
}

func TestResolveProjectRootGlobalFallsBackToCwd(t *testing.T) {
	oldGit := gitTopLevelFn
	defer func() { gitTopLevelFn = oldGit }()
	gitTopLevelFn = func() (string, error) { return "", errors.New("not a git repo") }

	// Pin the working directory so the assertion does not depend on whatever
	// directory other tests in the suite left the process in.
	workdir := t.TempDir()
	originalWd, _ := os.Getwd()
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(originalWd) })

	home, _ := os.UserHomeDir()
	paths := discover.SadrPaths{Root: filepath.Join(home, ".sadr"), IsGlobal: true}
	got := resolveProjectRoot(paths)

	// Compare resolved paths (temp dirs may live behind symlinks, e.g. macOS).
	wantWd, _ := filepath.EvalSymlinks(workdir)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != wantWd {
		t.Errorf("global fallback without git should resolve to cwd, got %q want %q", got, workdir)
	}
	// The core of the bug: the global fallback must never resolve to the home
	// directory (filepath.Dir of ~/.sadr), or the file picker lists the wrong
	// tree and diff auto-selection silently fails.
	if got == filepath.Dir(paths.Root) {
		t.Errorf("global fallback must not resolve to home dir %q", filepath.Dir(paths.Root))
	}
}
