package doctor

// AuditTarget pairs a changed file with the active record(s) that document it.
type AuditTarget struct {
	FileRef string   `json:"file_ref"`
	Records []string `json:"records"`
}

// FilterChangedFiles intersects the changed files with the file_refs of active
// records, returning one AuditTarget per changed-and-documented file (in the
// order the files appear in the diff). Changed files with no active record are
// not audited; records whose file is untouched are skipped.
func FilterChangedFiles(changed []string, records []RecordRef) []AuditTarget {
	byFileRef := map[string][]string{}
	for _, r := range records {
		if r.Status != StatusActive || r.FileRef == "" {
			continue
		}
		byFileRef[r.FileRef] = append(byFileRef[r.FileRef], r.ID)
	}

	var targets []AuditTarget
	seen := map[string]bool{}
	for _, f := range changed {
		if seen[f] {
			continue
		}
		if recs, ok := byFileRef[f]; ok {
			seen[f] = true
			targets = append(targets, AuditTarget{FileRef: f, Records: recs})
		}
	}
	return targets
}
