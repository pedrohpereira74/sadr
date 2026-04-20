package filepicker

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"
)

var skipDirs = map[string]bool{
	".git": true, ".sadr": true, "node_modules": true, "vendor": true,
	"__pycache__": true, ".venv": true, "venv": true, ".idea": true,
	".vscode": true, "dist": true, "build": true, ".next": true,
	"target": true, "bin": true, "obj": true,
}

var skipExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".o": true, ".a": true, ".pyc": true, ".class": true,
	".jar": true, ".lock": true,
}

var skipFiles = map[string]bool{
	"go.sum": true,
}

func ListProjectFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		name := d.Name()

		if d.IsDir() {
			if path == root {
				return nil
			}
			if skipDirs[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(name, ".") {
			return nil
		}
		if skipExtensions[filepath.Ext(name)] {
			return nil
		}
		if skipFiles[name] {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		files = append(files, rel)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func FilterFiles(files []string, query string) []string {
	if query == "" {
		return files
	}

	results := fuzzy.Find(query, files)
	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.Str
	}
	return out
}
