package discover

import (
	"fmt"
	"os"
	"path/filepath"
)

type SadrPaths struct {
	Root       string
	Records    string
	Exports    string
	Answers    string
	ConfigsDir string
	Username   string
	IsGlobal   bool
}

func FindSadrDir(startDir string) (SadrPaths, error) {
	dir := startDir

	for {
		candidate := filepath.Join(dir, ".sadr")
		if _, err := os.Stat(candidate); err == nil {
			paths := buildPaths(candidate)
			
			home, errHome := os.UserHomeDir()
			if errHome == nil && filepath.Clean(dir) == filepath.Clean(home) {
				paths.IsGlobal = true
			} else {
				paths.IsGlobal = false
			}

			return paths, nil
		}

		home, errHome := os.UserHomeDir()
		if errHome == nil && filepath.Clean(dir) == filepath.Clean(home) {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	home, err := os.UserHomeDir()
	if err == nil {
		global := filepath.Join(home, ".sadr")
		if _, err := os.Stat(global); err == nil {
			paths := buildPaths(global)
			paths.IsGlobal = true
			return paths, nil
		}
	}

	return SadrPaths{}, fmt.Errorf("no .sadr/ found. Run 'sadr init' to setup this repository")
}

func buildPaths(root string) SadrPaths {
	return SadrPaths{
		Root:       root,
		Records:    filepath.Join(root, "records"),
		Exports:    filepath.Join(root, "exports"),
		ConfigsDir: filepath.Join(root, "configs"),
	}
}
