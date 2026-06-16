package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/discover"
	"github.com/pedrohpereira74/sadr/internal/doctor"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
	"github.com/spf13/cobra"
)

type doctorOptions struct {
	ci    bool
	apply string
}

func parseApplyIDs(csv string) []string {
	var ids []string
	for part := range strings.SplitSeq(csv, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			ids = append(ids, part)
		}
	}
	return ids
}

func doctorActorRole() string {
	if v := os.Getenv("GITHUB_ACTOR_ASSOCIATION"); v != "" {
		return v
	}
	return os.Getenv("DOCTOR_ACTOR_ROLE")
}

func underRoot(p string) bool {
	if filepath.IsAbs(p) {
		return false
	}
	clean := filepath.Clean(p)
	return clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func gitCommitImpl(root string, paths []string, message string) error {
	rel := make([]string, 0, len(paths))
	for _, p := range paths {
		if r, err := filepath.Rel(root, p); err == nil {
			rel = append(rel, r)
		} else {
			rel = append(rel, p)
		}
	}
	addArgs := append([]string{"-C", root, "add"}, rel...)
	if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("doctor: git add failed: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", root, "commit", "-m", message).CombinedOutput(); err != nil {
		return fmt.Errorf("doctor: git commit failed: %v: %s", err, out)
	}
	return nil
}

func doctorRepoRoot() string {
	if root, err := gitTopLevelFn(); err == nil && root != "" {
		return root
	}
	dir, _ := os.Getwd()
	return dir
}

func doctorPaths() (discover.SadrPaths, bool) {
	paths, err := discover.FindSadrDir(doctorRepoRoot())
	if err != nil || paths.IsGlobal {
		return discover.SadrPaths{}, false
	}
	return paths, true
}

func entryID(e storage.RecordEntry) string {
	if e.Author != "" {
		return fmt.Sprintf("%s/%d", e.Author, e.FileID)
	}
	return fmt.Sprintf("#%d", e.FileID)
}

func recordRefsFromEntries(entries []storage.RecordEntry) []doctor.RecordRef {
	refs := make([]doctor.RecordRef, 0, len(entries))
	for _, e := range entries {
		refs = append(refs, doctor.RecordRef{
			ID:      entryID(e),
			FileRef: e.Record.FileRef,
			Status:  e.Record.Status,
			Related: e.Record.Related,
		})
	}
	return refs
}

func deactivateRefs(refs []doctor.RecordRef, deprecated map[string]bool) []doctor.RecordRef {
	out := make([]doctor.RecordRef, len(refs))
	for i, r := range refs {
		if deprecated[r.ID] {
			r.Status = doctor.StatusDeprecated
		}
		out[i] = r
	}
	return out
}

func indexEntries(entries []storage.RecordEntry) map[string]storage.RecordEntry {
	byID := make(map[string]storage.RecordEntry, len(entries))
	for _, e := range entries {
		byID[entryID(e)] = e
	}
	return byID
}

func deprecateRecords(ids []string, entries []storage.RecordEntry) ([]string, map[string]bool, error) {
	byID := indexEntries(entries)
	s := storage.NewStorage("")
	var changed []string
	done := map[string]bool{}

	for _, id := range ids {
		e, ok := byID[id]
		if !ok {
			continue
		}
		if e.Record.Status == doctor.StatusDeprecated {
			done[id] = true
			continue
		}
		e.Record.Status = doctor.StatusDeprecated
		if err := s.UpdateRecord(e.Path, e.Record); err != nil {
			return nil, nil, fmt.Errorf("doctor: failed to update %s: %w", e.Path, err)
		}
		changed = append(changed, e.Path)
		done[id] = true
	}

	sort.Strings(changed)
	return changed, done, nil
}

func newDoctorCmd() *cobra.Command {
	opts := &doctorOptions{}

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate records and flag conflicting records (CI gatekeeper)",
		Long: "Validate records repo-wide and flag any file documented by more than one\n" +
			"active record. Designed to run in CI (--ci) as a merge gatekeeper; resolve\n" +
			"conflicts by deprecating the stale record with --apply.",
		RunE:          runDoctor(opts),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolVar(&opts.ci, "ci", false, "Non-interactive CI mode with structured output for ChatOps")
	cmd.Flags().StringVar(&opts.apply, "apply", "", "Comma-separated record IDs to deprecate (triggers the apply phase)")
	return cmd
}

func runDoctor(opts *doctorOptions) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ids := parseApplyIDs(opts.apply)
		applying := len(ids) > 0

		if applying {
			if role := doctorActorRole(); role != "" && !doctor.IsAuthorized(role) {
				return fmt.Errorf("doctor: actor role %q is not authorized to apply fixes", role)
			}
		}

		root := doctorRepoRoot()
		paths, ok := doctorPaths()
		if !ok {
			ui.Success(os.Stderr, "doctor: no records to validate.")
			return nil
		}
		entries, err := listAllRecordEntries(paths.Root)
		if err != nil {
			return fmt.Errorf("doctor: unexpected error: %w", err)
		}

		fileExists := func(p string) bool {
			if !underRoot(p) {
				return false
			}
			_, statErr := os.Stat(filepath.Join(root, p))
			return statErr == nil
		}

		refs := recordRefsFromEntries(entries)
		result := doctor.Validate(refs, fileExists)
		for _, o := range result.Orphans {
			ui.Warning(os.Stderr, fmt.Sprintf("doctor: orphan record %s points at missing file %q", o.Record, o.FileRef))
		}

		if applying {
			changedPaths, deprecated, dErr := deprecateRecords(ids, entries)
			if dErr != nil {
				return dErr
			}
			if len(changedPaths) > 0 {
				msg := fmt.Sprintf("docs(sadr): doctor deprecated %d record(s)", len(changedPaths))
				if cErr := gitCommitFn(root, changedPaths, msg); cErr != nil {
					return cErr
				}
				ui.Success(os.Stderr, fmt.Sprintf("doctor: deprecated and committed %d record(s)", len(changedPaths)))
			}

			post := doctor.Validate(deactivateRefs(refs, deprecated), fileExists)
			if !post.OK() {
				return fmt.Errorf("doctor: %d conflict(s) and %d orphan(s) still unresolved after apply; merge blocked", len(post.Collisions), len(post.Orphans))
			}
			ui.Success(os.Stderr, "doctor: all conflicts resolved.")
			return nil
		}

		if !result.OK() {
			fmt.Fprintln(os.Stdout, doctor.RenderComment(result))
			return fmt.Errorf("doctor: conflicting and/or orphan records. %d conflict(s) and %d orphan(s) detected", len(result.Collisions), len(result.Orphans))
		}

		ui.Success(os.Stderr, "doctor: records are consistent.")
		return nil
	}
}

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}
