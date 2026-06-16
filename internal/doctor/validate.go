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
	Related []string
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

func Validate(records []RecordRef, fileExists func(string) bool) ValidationResult {
	var res ValidationResult
	byFileRef := map[string][]RecordRef{}
	related := map[string]map[string]bool{}

	for _, r := range records {
		if r.Status != StatusActive {
			continue
		}
		rel := make(map[string]bool, len(r.Related))
		for _, id := range r.Related {
			rel[id] = true
		}
		related[r.ID] = rel

		seenPath := map[string]bool{}
		for _, fr := range model.ParseTags(r.FileRef) {
			if fr == model.NoFileRef || seenPath[fr] {
				continue
			}
			seenPath[fr] = true
			if !fileExists(fr) {
				res.Orphans = append(res.Orphans, Orphan{Record: r.ID, FileRef: fr})
			}
			byFileRef[fr] = append(byFileRef[fr], r)
		}
	}

	keys := make([]string, 0, len(byFileRef))
	for k := range byFileRef {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		recs := byFileRef[k]
		if len(recs) < 2 {
			continue
		}
		sort.Slice(recs, func(i, j int) bool { return recs[i].ID < recs[j].ID })

		involved := map[string]bool{}
		var order []string
		for i := range recs {
			for j := i + 1; j < len(recs); j++ {
				a, b := recs[i].ID, recs[j].ID
				if related[a][b] || related[b][a] {
					continue
				}
				for _, id := range []string{a, b} {
					if !involved[id] {
						involved[id] = true
						order = append(order, id)
					}
				}
			}
		}
		if len(order) > 0 {
			sort.Strings(order)
			res.Collisions = append(res.Collisions, Collision{FileRef: k, Records: order})
		}
	}

	return res
}
