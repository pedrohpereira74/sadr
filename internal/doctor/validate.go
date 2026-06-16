package doctor

import (
	"sort"

	"github.com/pedrohpereira74/sadr/internal/model"
)

const (
	StatusActive     = "active"
	StatusDeprecated = "deprecated"
)

type RecordRef struct {
	ID      string
	FileRef string
	Status  string
}

type Orphan struct {
	Record  string `json:"record"`
	FileRef string `json:"file_ref"`
}

type Collision struct {
	FileRef string   `json:"file_ref"`
	Records []string `json:"records"`
}

type ValidationResult struct {
	Orphans    []Orphan    `json:"orphans"`
	Collisions []Collision `json:"collisions"`
}

func (r ValidationResult) OK() bool {
	return len(r.Orphans) == 0 && len(r.Collisions) == 0
}

func RemainingCollisions(collisions []Collision, deprecated map[string]bool) []Collision {
	var remaining []Collision
	for _, c := range collisions {
		var active []string
		for _, r := range c.Records {
			if !deprecated[r] {
				active = append(active, r)
			}
		}
		if len(active) > 1 {
			remaining = append(remaining, Collision{FileRef: c.FileRef, Records: active})
		}
	}
	return remaining
}

func RemainingOrphans(orphans []Orphan, deprecated map[string]bool) []Orphan {
	var remaining []Orphan
	for _, o := range orphans {
		if !deprecated[o.Record] {
			remaining = append(remaining, o)
		}
	}
	return remaining
}

func Validate(records []RecordRef, fileExists func(string) bool) ValidationResult {
	var res ValidationResult
	byFileRef := map[string][]string{}

	for _, r := range records {
		if r.Status != StatusActive {
			continue
		}
		seenPath := map[string]bool{}
		for _, fr := range model.ParseTags(r.FileRef) {
			if fr == model.NoFileRef || seenPath[fr] {
				continue
			}
			seenPath[fr] = true
			if !fileExists(fr) {
				res.Orphans = append(res.Orphans, Orphan{Record: r.ID, FileRef: fr})
			}
			byFileRef[fr] = append(byFileRef[fr], r.ID)
		}
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
