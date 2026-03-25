package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSadrDirInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".sadr", "records"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	paths, err := FindSadrDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(dir, ".sadr")
	if paths.Root != expected {
		t.Errorf("expected root '%s', got '%s'", expected, paths.Root)
	}
	if paths.Records != filepath.Join(expected, "records") {
		t.Errorf("expected records '%s', got '%s'", filepath.Join(expected, "records"), paths.Records)
	}
}

func TestFindSadrDirWalksUp(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".sadr", "records"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	deep := filepath.Join(root, "src", "internal", "model")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatalf("failed to create deep dir: %v", err)
	}

	paths, err := FindSadrDir(deep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(root, ".sadr")
	if paths.Root != expected {
		t.Errorf("expected root '%s', got '%s'", expected, paths.Root)
	}
}

func TestFindSadrDirFallsBackToGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if err := os.MkdirAll(filepath.Join(home, ".sadr", "records"), 0755); err != nil {
		t.Fatalf("failed to create global dir: %v", err)
	}

	noProject := filepath.Join(home, "projects", "proj1")
	os.MkdirAll(noProject, 0755)
	
	paths, err := FindSadrDir(noProject)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(home, ".sadr")
	if paths.Root != expected {
		t.Errorf("expected global '%s', got '%s'", expected, paths.Root)
	}
}

func TestFindSadrDirReturnsErrorWhenNoneFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	noProject := filepath.Join(home, "projects", "proj2")
	os.MkdirAll(noProject, 0755)
	
	_, err := FindSadrDir(noProject)
	if err == nil {
		t.Fatal("expected error when no .sadr/ found anywhere, got nil")
	}
}
