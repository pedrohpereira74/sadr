package search

import "strings"

func HasAnyTag(recordTags string, filterTags string) bool {
	rTags := strings.Split(recordTags, ",")
	fTags := strings.Split(filterTags, ",")

	for _, ft := range fTags {
		ft = strings.TrimSpace(ft)
		for _, rt := range rTags {
			if strings.TrimSpace(rt) == ft {
				return true
			}
		}
	}
	return false
}
