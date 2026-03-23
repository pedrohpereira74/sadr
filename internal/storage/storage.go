package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedrohpereira74/sadr/internal/model"
	"gopkg.in/yaml.v3"
)

type Storage struct {
	Dir string
}

func NewStorage(dir string) *Storage {
	return &Storage{Dir: dir}
}

func (s *Storage) SaveRecord(r model.Record) (string, error) {
	frontmatter := map[string]interface{}{
		"schema_version": model.SchemaVersion,
		"type":           r.Type,
		"title":          r.Title,
		"file_ref":       r.FileRef,
		"created":        r.CreatedAt.Format(time.RFC3339),
	}

	frontmatterKeys := []string{"tags"}
	for _, key := range frontmatterKeys {
		if value, exists := r.Fields[key]; exists && value != "" {
			if key == "tags" {
				raw := strings.Split(value, ",")
				var clean []string
				for _, t := range raw {
					t = strings.TrimSpace(t)
					if t != "" {
						clean = append(clean, t)
					}
				}
				frontmatter[key] = clean
			} else {
				frontmatter[key] = value
			}
		}
	}

	yamlBytes, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", err
	}

	var content strings.Builder
	content.WriteString("---\n")
	content.Write(yamlBytes)
	content.WriteString("---\n\n")

	content.WriteString(fmt.Sprintf("# %s\n\n", r.Title))

	if tagsStr, ok := r.Fields["tags"]; ok && tagsStr != "" {
		tagsList := strings.Split(tagsStr, ",")
		var formattedTags []string
		for _, t := range tagsList {
			formattedTags = append(formattedTags, "`#"+strings.TrimSpace(t)+"`")
		}
		content.WriteString(fmt.Sprintf("**Tags:** %s\n\n", strings.Join(formattedTags, " ")))
	}

	if r.Snippet != "" {
		content.WriteString("## Snippet\n\n```\n")
		content.WriteString(r.Snippet)
		content.WriteString("\n```\n\n")
	}

	written := map[string]bool{"tags": true}

	for _, key := range r.FieldOrder {
		value, ok := r.Fields[key]
		if !ok || value == "" || written[key] {
			continue
		}

		capitalizedKey := key
		if len(key) > 0 {
			capitalizedKey = strings.ToUpper(key[:1]) + strings.ToLower(key[1:])
		}
		content.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", capitalizedKey, strings.TrimSpace(value)))
		written[key] = true
	}

	for key, value := range r.Fields {
		if value == "" || written[key] {
			continue
		}

		capitalizedKey := key
		if len(key) > 0 {
			capitalizedKey = strings.ToUpper(key[:1]) + strings.ToLower(key[1:])
		}
		content.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", capitalizedKey, strings.TrimSpace(value)))
	}

	slug := slugify(r.Title)
	nextID := s.getMaxID() + 1
	filename := fmt.Sprintf("sadr-%04d-%s.md", nextID, slug)
	fullPath := filepath.Join(s.Dir, filename)

	err = os.WriteFile(fullPath, []byte(content.String()), 0644)
	if err != nil {
		return "", err
	}

	return fullPath, nil
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

	schemaVersion, _ := front["schema_version"].(int)
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
	sections := splitSections(body)
	for name, value := range sections {
		if strings.ToLower(name) == "snippet" {
			r.Snippet = extractCodeBlock(value)
		} else {
			r.Fields[strings.ToLower(name)] = strings.TrimSpace(value)
		}
	}

	return r, nil
}

func (s *Storage) ListRecords() ([]model.Record, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}

	var records []model.Record
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.Dir, entry.Name())
		r, err := s.LoadRecord(path)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

func (s *Storage) DeleteRecord(path string) error {
	return os.Remove(path)
}

func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	return s
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

func splitSections(body string) map[string]string {
	sections := map[string]string{}
	current := ""
	var buf strings.Builder

	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			if current != "" {
				sections[current] = buf.String()
			}
			current = strings.TrimPrefix(line, "## ")
			buf.Reset()
		} else {
			buf.WriteString(line + "\n")
		}
	}
	if current != "" {
		sections[current] = buf.String()
	}

	return sections
}

func extractCodeBlock(s string) string {
	start := strings.Index(s, "```")
	if start == -1 {
		return strings.TrimSpace(s)
	}
	afterStart := s[start+3:]
	newline := strings.Index(afterStart, "\n")
	if newline == -1 {
		return ""
	}
	code := afterStart[newline+1:]
	end := strings.Index(code, "```")
	if end == -1 {
		return strings.TrimSpace(code)
	}
	return strings.TrimSpace(code[:end])
}
