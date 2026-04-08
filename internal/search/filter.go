package search

import "strings"

func HasAnyTag(recordTags string, filterTags string) bool {
	if strings.TrimSpace(recordTags) == "" || strings.TrimSpace(filterTags) == "" {
		return false
	}
	for ft := range strings.SplitSeq(filterTags, ",") {
		ft = strings.TrimSpace(ft)
		for rt := range strings.SplitSeq(recordTags, ",") {
			if strings.EqualFold(strings.TrimSpace(rt), ft) {
				return true
			}
		}
	}
	return false
}
