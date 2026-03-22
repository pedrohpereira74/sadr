package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pedrohpereira74/sadr/internal/discover"
)

func findEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	for _, fallback := range []string{"vim", "nano", "vi"} {
		if _, err := exec.LookPath(fallback); err == nil {
			return fallback
		}
	}
	return ""
}

func openEditor(editor string, path string) {
	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, ":(  Editor exited with error: %v\n", err)
	}
}

func resolveRecordsDir(global bool) (string, error) {
	if global {
		home, _ := os.UserHomeDir()
		recordsDir := filepath.Join(home, ".sadr", "records")
		if _, err := os.Stat(recordsDir); os.IsNotExist(err) {
			return "", fmt.Errorf("Global storage not found. Run 'sadr config --global' first")
		}
		return recordsDir, nil
	}

	dir, _ := os.Getwd()
	paths, err := discover.FindSadrDir(dir)
	if err != nil {
		return "", err
	}
	return paths.Records, nil
}

func resolvePaths(global bool) (discover.SadrPaths, error) {
	if global {
		home, _ := os.UserHomeDir()
		root := filepath.Join(home, ".sadr")
		if _, err := os.Stat(root); os.IsNotExist(err) {
			return discover.SadrPaths{}, fmt.Errorf("Global storage not found. Run 'sadr config --global' first")
		}
		return discover.SadrPaths{
			Root:    root,
			Records: filepath.Join(root, "records"),
			Exports: filepath.Join(root, "exports"),
		}, nil
	}

	dir, _ := os.Getwd()
	return discover.FindSadrDir(dir)
}
