package search

import (
	"strings"

	"github.com/pedrohpereira74/sadr/internal/model"
)

func Matches(r model.Record, query string, deep bool) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(r.Title), q) {
		return true
	}
	if strings.Contains(strings.ToLower(r.Fields["tags"]), q) {
		return true
	}
	if deep {
		if strings.Contains(strings.ToLower(r.Snippet), q) {
			return true
		}
		for key, value := range r.Fields {
			if key == "tags" {
				continue
			}
			if strings.Contains(strings.ToLower(value), q) {
				return true
			}
		}
	}
	return false
}

func Search(records []model.Record, query string, deep bool) []model.Record {
	if query == "" {
		return records
	}
	var results []model.Record
	for _, r := range records {
		if Matches(r, query, deep) {
			results = append(results, r)
		}
	}
	if results == nil {
		return []model.Record{}
	}
	return results
}
