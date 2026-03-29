package search

import (
	"strings"

	"github.com/pedrohpereira74/sadr/internal/model"
)

func Search(records []model.Record, query string, deep bool) []model.Record {
	query = strings.ToLower(query)
	var results []model.Record

	for _, r := range records {
		title := strings.ToLower(r.Title)
		tags := strings.ToLower(r.Fields["tags"])

		if strings.Contains(title, query) || strings.Contains(tags, query) {
			results = append(results, r)
			continue
		}

		if deep {
			found := false
			if strings.Contains(strings.ToLower(r.Snippet), query) {
				found = true
			}
			if !found {
				for _, value := range r.Fields {
					if strings.Contains(strings.ToLower(value), query) {
						found = true
						break
					}
				}
			}
			if found {
				results = append(results, r)
			}
		}
	}

	return results
}
