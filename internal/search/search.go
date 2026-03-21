package search

import (
	"strings"

	"github.com/pedrohpereira74/sadr/internal/model"
	"github.com/pedrohpereira74/sadr/internal/storage"
)

func Search(dir string, query string, deep bool) ([]model.Record, error) {
	s := storage.NewStorage(dir)

	records, err := s.ListRecords()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []model.Record

	for _, r := range records {
		// Sempre busca em título e tags
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

	return results, nil
}
