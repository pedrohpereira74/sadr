package filepicker

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T, files []string) string {
	t.Helper()
	dir := t.TempDir()

	for _, f := range files {
		path := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	return dir
}

func TestListProjectFiles(t *testing.T) {
	dir := setupTestDir(t, []string{
		"main.go",
		"src/handlers/payment.go",
		"src/handlers/auth.go",
		"src/models/user.go",
		".git/config",
		".sadr/records/001.yaml",
		"node_modules/pkg/index.js",
		"go.sum",
	})

	files, err := ListProjectFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"main.go",
		"src/handlers/auth.go",
		"src/handlers/payment.go",
		"src/models/user.go",
	}

	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}

	for i, f := range expected {
		if files[i] != f {
			t.Errorf("file %d: expected '%s', got '%s'", i, f, files[i])
		}
	}
}

func TestListProjectFilesEmpty(t *testing.T) {
	dir := t.TempDir()

	files, err := ListProjectFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestListProjectFilesIgnoresHidden(t *testing.T) {
	dir := setupTestDir(t, []string{
		"visible.go",
		".hidden_file",
		".hidden_dir/something.go",
	})

	files, err := ListProjectFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(files), files)
	}
	if files[0] != "visible.go" {
		t.Errorf("expected 'visible.go', got '%s'", files[0])
	}
}

func TestListProjectFilesIgnoresBinaries(t *testing.T) {
	dir := setupTestDir(t, []string{
		"main.go",
		"app.exe",
		"lib.dll",
		"lib.so",
		"module.pyc",
	})

	files, err := ListProjectFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(files), files)
	}
	if files[0] != "main.go" {
		t.Errorf("expected 'main.go', got '%s'", files[0])
	}
}

func TestFilterFiles(t *testing.T) {
	files := []string{
		"src/handlers/payment.go",
		"src/handlers/auth.go",
		"src/models/user.go",
	}

	result := FilterFiles(files, "payment")
	if len(result) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result))
	}
	if result[0] != "src/handlers/payment.go" {
		t.Errorf("expected 'src/handlers/payment.go', got '%s'", result[0])
	}
}

func TestFilterFilesPath(t *testing.T) {
	files := []string{
		"src/handlers/payment.go",
		"src/handlers/auth.go",
		"src/models/user.go",
	}

	result := FilterFiles(files, "handlers/")
	if len(result) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result))
	}
}

func TestFilterFilesCaseInsensitive(t *testing.T) {
	files := []string{
		"src/handlers/payment.go",
		"src/handlers/auth.go",
	}

	result := FilterFiles(files, "PAYMENT")
	if len(result) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result))
	}
	if result[0] != "src/handlers/payment.go" {
		t.Errorf("expected 'src/handlers/payment.go', got '%s'", result[0])
	}
}

func TestFilterFilesEmpty(t *testing.T) {
	files := []string{
		"src/handlers/payment.go",
		"src/handlers/auth.go",
		"src/models/user.go",
	}

	result := FilterFiles(files, "")
	if len(result) != len(files) {
		t.Errorf("expected %d files, got %d", len(files), len(result))
	}
}
