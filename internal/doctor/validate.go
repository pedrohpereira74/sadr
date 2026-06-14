package doctor

import (
	"sort"

	"github.com/pedrohpereira74/sadr/internal/model"
)

// StatusActive is the canonical status of a record that is the current source
// of truth (set on creation; see cmd/new.go).
const StatusActive = "active"

// RecordRef is the minimal view of a record the validator needs.
type RecordRef struct {
	ID      string // human label, e.g. "alice/3"
	FileRef string
	Status  string
}

// Orphan is an active record whose file_ref no longer exists on disk.
type Orphan struct {
	Record  string `json:"record"`
	FileRef string `json:"file_ref"`
}

// Collision is a file_ref pointed at by more than one active record.
type Collision struct {
	FileRef string   `json:"file_ref"`
	Records []string `json:"records"`
}

// ValidationResult is the deterministic outcome of record validation.
type ValidationResult struct {
	Orphans    []Orphan    `json:"orphans"`
	Collisions []Collision `json:"collisions"`
}

// OK reports whether validation found no problems.
func (r ValidationResult) OK() bool {
	return len(r.Orphans) == 0 && len(r.Collisions) == 0
}

// Validate checks active records deterministically: every file_ref other than
// "N/A" must exist (fileExists predicate), and no two active records may share
// the same file_ref. Records with a non-active status are ignored.
func Validate(records []RecordRef, fileExists func(string) bool) ValidationResult {
	var res ValidationResult
	byFileRef := map[string][]string{}

	for _, r := range records {
		if r.Status != StatusActive {
			continue
		}
		if r.FileRef == "" || r.FileRef == model.NoFileRef {
			continue
		}
		if !fileExists(r.FileRef) {
			res.Orphans = append(res.Orphans, Orphan{Record: r.ID, FileRef: r.FileRef})
		}
		byFileRef[r.FileRef] = append(byFileRef[r.FileRef], r.ID)
	}

	keys := make([]string, 0, len(byFileRef))
	for k := range byFileRef {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if len(byFileRef[k]) > 1 {
			res.Collisions = append(res.Collisions, Collision{FileRef: k, Records: byFileRef[k]})
		}
	}

	return res
}
