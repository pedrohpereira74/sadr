package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/pedrohpereira74/sadr/internal/model"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"gopkg.in/yaml.v3"
)

var (
	invalidSlugChars = regexp.MustCompile(`[^a-z0-9\-]`)
	multipleDashes   = regexp.MustCompile(`-{2,}`)
)

type Storage struct {
	Dir string
}

func NewStorage(dir string) *Storage {
	return &Storage{Dir: dir}
}

func (s *Storage) SaveRecord(r model.Record) (string, error) {
	frontmatter := buildFrontmatter(r)
	yamlBytes, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", err
	}

	var content strings.Builder
	content.WriteString("---\n")
	content.Write(yamlBytes)
	content.WriteString("---\n\n")

	formatBody(&content, r)

	slug := Slugify(r.Title)
	nextID := s.getMaxID() + 1
	filename := fmt.Sprintf("sadr-%04d-%s.md", nextID, slug)
	fullPath := filepath.Join(s.Dir, filename)

	if err := os.WriteFile(fullPath, []byte(content.String()), 0644); err != nil {
		return "", err
	}

	return fullPath, nil
}

func buildFrontmatter(r model.Record) map[string]interface{} {
	fm := map[string]interface{}{
		"schema_version": model.SchemaVersion,
		"type":           r.Type,
		"title":          r.Title,
		"file_ref":       r.FileRef,
		"created":        r.CreatedAt.Format(time.RFC3339),
	}

	if tagsStr, ok := r.Fields["tags"]; ok && tagsStr != "" {
		raw := strings.Split(tagsStr, ",")
		var clean []string
		for _, t := range raw {
			t = strings.TrimSpace(t)
			if t != "" {
				clean = append(clean, t)
			}
		}
		fm["tags"] = clean
	}

	return fm
}

func formatBody(content *strings.Builder, r model.Record) {
	fmt.Fprintf(content, "# %s\n\n", r.Title)

	if tagsStr, ok := r.Fields["tags"]; ok && tagsStr != "" {
		tagsList := strings.Split(tagsStr, ",")
		var formattedTags []string
		for _, t := range tagsList {
			formattedTags = append(formattedTags, "`#"+strings.TrimSpace(t)+"`")
		}
		fmt.Fprintf(content, "**Tags:** %s\n\n", strings.Join(formattedTags, " "))
	}

	if r.Snippet != "" {
		bt := determineBackticks(r.Snippet)
		fmt.Fprintf(content, "## Snippet\n\n%s\n%s\n%s\n\n", bt, strings.TrimSpace(r.Snippet), bt)
	}

	written := map[string]bool{"tags": true}

	for _, key := range r.FieldOrder {
		value, ok := r.Fields[key]
		if !ok || value == "" || written[key] {
			continue
		}
		fmt.Fprintf(content, "## %s\n\n%s\n\n", capitalizeKey(key), strings.TrimSpace(value))
		written[key] = true
	}

	for key, value := range r.Fields {
		if value == "" || written[key] {
			continue
		}
		fmt.Fprintf(content, "## %s\n\n%s\n\n", capitalizeKey(key), strings.TrimSpace(value))
	}
}

func (s *Storage) LoadRecord(path string) (model.Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Record{}, err
	}

	content := string(data)
	parts := strings.SplitN(content, "---", 3)
	if len(parts) != 3 {
		return model.Record{}, fmt.Errorf("invalid record format: missing frontmatter")
	}

	var front map[string]interface{}
	err = yaml.Unmarshal([]byte(parts[1]), &front)
	if err != nil {
		return model.Record{}, err
	}

	createdStr, _ := front["created"].(string)
	createdAt, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return model.Record{}, fmt.Errorf("invalid created timestamp: %v", err)
	}

	var schemaVersion int
	switch sv := front["schema_version"].(type) {
	case int:
		schemaVersion = sv
	case float64:
		schemaVersion = int(sv)
	case int64:
		schemaVersion = int(sv)
	}
	title, _ := front["title"].(string)
	recordType, _ := front["type"].(string)
	fileRef, _ := front["file_ref"].(string)

	r := model.Record{
		Title:         title,
		Type:          recordType,
		SchemaVersion: schemaVersion,
		FileRef:       fileRef,
		CreatedAt:     createdAt,
		Fields:        map[string]string{},
	}

	if tags, ok := front["tags"].([]interface{}); ok {
		strs := make([]string, len(tags))
		for i, v := range tags {
			strs[i] = fmt.Sprintf("%v", v)
		}
		r.Fields["tags"] = strings.Join(strs, ",")
	}

	body := strings.TrimSpace(parts[2])
	sections, sectionOrder := splitSections(body)
	for name, value := range sections {
		if strings.ToLower(name) == "snippet" {
			r.Snippet = extractCodeBlock(value)
		} else {
			r.Fields[strings.ToLower(name)] = strings.TrimSpace(value)
		}
	}

	for _, name := range sectionOrder {
		lower := strings.ToLower(name)
		if lower != "snippet" {
			r.FieldOrder = append(r.FieldOrder, lower)
		}
	}

	return r, nil
}

func (s *Storage) ListRecords() ([]model.Record, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	var records []model.Record
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.Dir, entry.Name())
		r, err := s.LoadRecord(path)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, ":(  Skipping invalid record %s: %v\n", entry.Name(), err)
			continue
		}
		records = append(records, r)
	}
	return records, nil
}

func (s *Storage) DeleteRecord(path string) error {
	return os.Remove(path)
}

func capitalizeKey(key string) string {
	if len(key) == 0 {
		return key
	}
	runes := []rune(key)
	return strings.ToUpper(string(runes[0])) + strings.ToLower(string(runes[1:]))
}

func Slugify(title string) string {
	s := strings.ToLower(title)
	
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ = transform.String(t, s)

	s = strings.ReplaceAll(s, " ", "-")
	s = invalidSlugChars.ReplaceAllString(s, "")
	s = multipleDashes.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func (s *Storage) getMaxID() int {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return 0
	}
	maxID := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "sadr-") && strings.HasSuffix(entry.Name(), ".md") {
			parts := strings.Split(entry.Name(), "-")
			if len(parts) >= 2 {
				var id int
				_, err := fmt.Sscanf(parts[1], "%d", &id)
				if err == nil && id > maxID {
					maxID = id
				}
			}
		}
	}
	return maxID
}

func (s *Storage) GetRecordPathByIndex(idx int) (string, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return "", err
	}
	var mdFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			mdFiles = append(mdFiles, entry.Name())
		}
	}
	if idx < 0 || idx >= len(mdFiles) {
		return "", fmt.Errorf("record bounds error")
	}
	return filepath.Join(s.Dir, mdFiles[idx]), nil
}

func determineBackticks(snippet string) string {
	count := 3
	for strings.Contains(snippet, strings.Repeat("`", count)) {
		count++
	}
	return strings.Repeat("`", count)
}

func splitSections(body string) (map[string]string, []string) {
	sections := map[string]string{}
	var order []string
	current := ""
	var buf strings.Builder

	inCodeBlock := false
	codeBlockFence := ""

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)

		if fence := extractFence(trimmed); fence != "" {
			if !inCodeBlock {
				inCodeBlock = true
				codeBlockFence = fence
			} else if fence == codeBlockFence {
				inCodeBlock = false
			}
		}

		if !inCodeBlock && strings.HasPrefix(line, "## ") {
			if current != "" {
				sections[current] = buf.String()
			}
			current = strings.TrimPrefix(line, "## ")
			order = append(order, current)
			buf.Reset()
		} else {
			if current != "" {
				buf.WriteString(line + "\n")
			}
		}
	}
	if current != "" {
		sections[current] = buf.String()
	}

	return sections, order
}

func extractCodeBlock(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) >= 2 {
		first := strings.TrimSpace(lines[0])
		last := strings.TrimSpace(lines[len(lines)-1])
		firstFence := extractFence(first)
		lastFence := extractFence(last)
		if firstFence != "" && lastFence != "" && firstFence == lastFence {
			return strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	return strings.TrimSpace(s)
}

func extractFence(line string) string {
	count := 0
	for _, r := range line {
		if r == '`' {
			count++
		} else {
			break
		}
	}
	if count >= 3 {
		return strings.Repeat("`", count)
	}
	return ""
}
