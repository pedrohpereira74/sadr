package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
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
	mu  sync.Mutex
}

type RecordEntry struct {
	Record model.Record
	FileID int
	Path   string
	Author string
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
	fmt.Fprintf(&content, "---\n")
	content.Write(yamlBytes)
	fmt.Fprintf(&content, "---\n\n")

	formatBody(&content, r)

	slug := Slugify(r.Title)
	data := []byte(content.String())

	s.mu.Lock()
	defer s.mu.Unlock()

	nextID := s.getMaxID() + 1

	for range 100 {
		filename := fmt.Sprintf("sadr-record-%04d-%s.md", nextID, slug)
		fullPath := filepath.Join(s.Dir, filename)
		f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			if os.IsExist(err) {
				nextID++
				continue
			}
			return "", err
		}
		_, writeErr := f.Write(data)
		closeErr := f.Close()
		if writeErr != nil {
			_ = os.Remove(fullPath)
			return "", writeErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return fullPath, nil
	}

	return "", fmt.Errorf("failed to create record file after 100 attempts")
}

func buildFrontmatter(r model.Record) map[string]any {
	fm := map[string]any{
		"schema_version": model.SchemaVersion,
		"type":           r.Type,
		"title":          r.Title,
		"file_ref":       r.FileRef,
		"created":        r.CreatedAt.Format(time.RFC3339),
	}

	if r.Author != "" {
		fm["author"] = r.Author
	}

	if tagsStr, ok := r.Fields[model.FieldTags]; ok && tagsStr != "" {
		fm[model.FieldTags] = model.ParseTags(tagsStr)
	}

	if statusStr, ok := r.Fields[model.FieldStatus]; ok && statusStr != "" {
		fm[model.FieldStatus] = statusStr
	}

	return fm
}

func formatBody(content *strings.Builder, r model.Record) {
	fmt.Fprintf(content, "# %s\n\n", r.Title)

	if tagsStr, ok := r.Fields[model.FieldTags]; ok && tagsStr != "" {
		var formattedTags []string
		for _, t := range model.ParseTags(tagsStr) {
			formattedTags = append(formattedTags, "`#"+t+"`")
		}
		fmt.Fprintf(content, "**Tags:** %s\n\n", strings.Join(formattedTags, " "))
	}

	if statusStr, ok := r.Fields[model.FieldStatus]; ok && statusStr != "" {
		fmt.Fprintf(content, "**Status:** %s\n\n", statusStr)
	}

	if r.Snippet != "" {
		bt := determineBackticks(r.Snippet)
		fmt.Fprintf(content, "## Snippet\n\n%s\n%s\n%s\n\n", bt, strings.TrimSpace(r.Snippet), bt)
	}

	written := map[string]bool{model.FieldTags: true, model.FieldStatus: true}

	for _, key := range r.FieldOrder {
		value, ok := r.Fields[key]
		if !ok || value == "" || written[key] {
			continue
		}
		fmt.Fprintf(content, "## %s\n\n%s\n\n", capitalizeKey(key), strings.TrimSpace(value))
		written[key] = true
	}

	var remaining []string
	for key := range r.Fields {
		if !written[key] && r.Fields[key] != "" {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		fmt.Fprintf(content, "## %s\n\n%s\n\n", capitalizeKey(key), strings.TrimSpace(r.Fields[key]))
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

	var front map[string]any
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

	if schemaVersion > model.SchemaVersion {
		return model.Record{}, fmt.Errorf("record was created with a newer version of sadr (schema %d > %d)", schemaVersion, model.SchemaVersion)
	}

	title, _ := front["title"].(string)
	recordType, _ := front["type"].(string)
	fileRef, _ := front["file_ref"].(string)
	author, _ := front["author"].(string)

	validTypes := map[string]bool{"full": true, "snippet": true, "adr": true}
	if !validTypes[recordType] {
		_, _ = fmt.Fprintf(os.Stderr, "warning: record has unknown type '%s'\n", recordType)
	}

	r := model.Record{
		Title:         title,
		Type:          recordType,
		SchemaVersion: schemaVersion,
		FileRef:       fileRef,
		Author:        author,
		CreatedAt:     createdAt,
		Fields:        map[string]string{},
	}

	if tags, ok := front[model.FieldTags].([]any); ok {
		strs := make([]string, len(tags))
		for i, v := range tags {
			strs[i] = fmt.Sprintf("%v", v)
		}
		r.Fields[model.FieldTags] = strings.Join(strs, ",")
	}

	if status, ok := front[model.FieldStatus].(string); ok && status != "" {
		r.Fields[model.FieldStatus] = status
	}

	body := strings.TrimSpace(parts[2])
	sections, sectionOrder := splitSections(body)
	for name, value := range sections {
		normalized := strings.ToLower(name)
		normalized = strings.ReplaceAll(normalized, " ", "_")
		if normalized == model.FieldSnippet {
			r.Snippet = extractCodeBlock(value)
		} else {
			r.Fields[normalized] = strings.TrimSpace(value)
		}
	}

	for _, name := range sectionOrder {
		normalized := strings.ToLower(name)
		normalized = strings.ReplaceAll(normalized, " ", "_")
		if normalized != model.FieldSnippet {
			r.FieldOrder = append(r.FieldOrder, normalized)
		}
	}

	return r, nil
}

func (s *Storage) ListRecordEntries() ([]RecordEntry, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	var records []RecordEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.Dir, entry.Name())
		r, err := s.LoadRecord(path)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Skipping invalid record %s: %v\n", entry.Name(), err)
			continue
		}
		records = append(records, RecordEntry{
			Record: r,
			FileID: ParseFileID(entry.Name()),
			Path:   path,
			Author: r.Author,
		})
	}
	return records, nil
}

func (s *Storage) ListRecords() ([]model.Record, error) {
	entries, err := s.ListRecordEntries()
	if err != nil {
		return nil, err
	}
	var records []model.Record
	for _, e := range entries {
		records = append(records, e.Record)
	}
	return records, nil
}

func (s *Storage) GetRecordByFileID(id int) (model.Record, string, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return model.Record{}, "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") && ParseFileID(entry.Name()) == id {
			path := filepath.Join(s.Dir, entry.Name())
			r, err := s.LoadRecord(path)
			if err != nil {
				return model.Record{}, "", err
			}
			return r, path, nil
		}
	}
	return model.Record{}, "", fmt.Errorf("record #%d not found", id)
}

func (s *Storage) DeleteRecord(path string) error {
	return os.Remove(path)
}

func ParseFileID(filename string) int {
	if !strings.HasPrefix(filename, "sadr-") {
		return 0
	}
	parts := strings.SplitN(filename, "-", 4)
	if len(parts) < 3 {
		return 0
	}
	idPart := parts[2]
	if isAllDigits(parts[1]) {
		idPart = parts[1]
	}
	var id int
	if _, err := fmt.Sscanf(idPart, "%d", &id); err != nil {
		return 0
	}
	return id
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func capitalizeKey(key string) string {
	if len(key) == 0 {
		return key
	}
	words := strings.FieldsFunc(key, func(r rune) bool {
		return r == '_' || r == ' '
	})
	for i, w := range words {
		if len(w) > 0 {
			r := []rune(w)
			words[i] = strings.ToUpper(string(r[0])) + string(r[1:])
		}
	}
	return strings.Join(words, " ")
}

func Slugify(title string) string {
	s := strings.ToLower(title)

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ = transform.String(t, s)

	s = strings.ReplaceAll(s, " ", "-")
	s = invalidSlugChars.ReplaceAllString(s, "")
	s = multipleDashes.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "untitled"
	}

	const maxSlugLen = 80
	if len(s) > maxSlugLen {
		s = s[:maxSlugLen]
		s = strings.TrimRight(s, "-")
	}

	return s
}

func (s *Storage) getMaxID() int {
	return NextID(s.Dir) - 1
}

func NextID(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 1
	}
	maxID := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "sadr-") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if id := ParseFileID(entry.Name()); id > maxID {
			maxID = id
		}
	}
	return maxID + 1
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

	for line := range strings.SplitSeq(body, "\n") {
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
				fmt.Fprintf(&buf, "%s\n", line)
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
