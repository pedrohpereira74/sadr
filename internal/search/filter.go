package search

import "strings"

func HasAnyTag(recordTags []string, filterTags string) bool {
	if len(recordTags) == 0 || strings.TrimSpace(filterTags) == "" {
		return false
	}
	for ft := range strings.SplitSeq(filterTags, ",") {
		ft = strings.TrimSpace(ft)
		for _, rt := range recordTags {
			if strings.EqualFold(rt, ft) {
				return true
			}
		}
	}
	return false
}
