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
	base  string
	apply string
}

// parseApplyIDs splits the --apply CSV into trimmed, non-empty record IDs.
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

// gitDiffImpl returns the unified diff between the merge base of <base> and HEAD.
func gitDiffImpl(base string) (string, error) {
	out, err := exec.Command("git", "--no-pager", "diff", base+"...HEAD").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// doctorActorRole returns the role of the actor triggering an apply, from the
// GitHub author_association or a platform-neutral DOCTOR_ACTOR_ROLE (used by the
// GitLab job). Empty means the CI platform already gated the trigger natively.
func doctorActorRole() string {
	if v := os.Getenv("GITHUB_ACTOR_ASSOCIATION"); v != "" {
		return v
	}
	return os.Getenv("DOCTOR_ACTOR_ROLE")
}

// gitCommitImpl stages the given paths and commits them in the repo at root.
func gitCommitImpl(root string, paths []string, message string) error {
	addArgs := append([]string{"-C", root, "add"}, paths...)
	if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", root, "commit", "-m", message).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %v: %s", err, out)
	}
	return nil
}

// collectDoctorDiff returns the raw diff against base and the list of changed files.
func collectDoctorDiff(base string) (string, []string, error) {
	diff, err := gitDiffFn(base)
	if err != nil {
		return "", nil, fmt.Errorf("git diff against %q failed: %w", base, err)
	}
	return diff, extractFilesFromDiff(diff), nil
}

// doctorRepoRoot returns the git repository root the command runs in, falling
// back to the working directory.
func doctorRepoRoot() string {
	if root, err := gitTopLevelFn(); err == nil && root != "" {
		return root
	}
	dir, _ := os.Getwd()
	return dir
}

// doctorPaths resolves the project vault without the interactive global
// fallback — doctor is a CI gatekeeper and must run inside a sadr-tracked repo.
func doctorPaths() (discover.SadrPaths, error) {
	root := doctorRepoRoot()
	paths, err := discover.FindSadrDir(root)
	if err != nil {
		return paths, err
	}
	if paths.IsGlobal {
		return paths, fmt.Errorf("no project .sadr/ found in %s; doctor runs inside a sadr-tracked repository", root)
	}
	return paths, nil
}

// entryID is the human-facing label of a record entry, e.g. "alice/3".
func entryID(e storage.RecordEntry) string {
	if e.Author != "" {
		return fmt.Sprintf("%s/%d", e.Author, e.FileID)
	}
	return fmt.Sprintf("#%d", e.FileID)
}

// recordRefsFromEntries maps storage entries to the validator's minimal view.
func recordRefsFromEntries(entries []storage.RecordEntry) []doctor.RecordRef {
	refs := make([]doctor.RecordRef, 0, len(entries))
	for _, e := range entries {
		refs = append(refs, doctor.RecordRef{
			ID:      entryID(e),
			FileRef: e.Record.FileRef,
			Status:  e.Record.Status,
		})
	}
	return refs
}

// indexEntries maps each record entry by its human-facing ID (e.g. "alice/3").
func indexEntries(entries []storage.RecordEntry) map[string]storage.RecordEntry {
	byID := make(map[string]storage.RecordEntry, len(entries))
	for _, e := range entries {
		byID[entryID(e)] = e
	}
	return byID
}

// deprecateRecords sets status=deprecated on the records named by ids, writing
// each in place. It returns the changed file paths (sorted) and the set of ids
// actually deprecated (unknown ids are skipped).
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
			return nil, nil, fmt.Errorf("failed to update %s: %w", e.Path, err)
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
		Short: "Audit records against the diff (CI gatekeeper)",
		Long: "Validate records and flag changed files documented by conflicting records.\n" +
			"Designed to run in CI (--ci) as a merge gatekeeper; resolve conflicts by\n" +
			"deprecating the stale record with --apply.",
		RunE:          runDoctor(opts),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolVar(&opts.ci, "ci", false, "Non-interactive CI mode with structured output for ChatOps")
	cmd.Flags().StringVar(&opts.base, "base", "main", "Base branch of the pull request")
	cmd.Flags().StringVar(&opts.apply, "apply", "", "Comma-separated record IDs to deprecate (triggers the apply phase)")
	return cmd
}

func runDoctor(opts *doctorOptions) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ids := parseApplyIDs(opts.apply)
		applying := len(ids) > 0

		// Defense in depth: the CI job is the first authorization gate (GitHub
		// issue_comment / GitLab manual job), but if it passes an actor role,
		// refuse unauthorized applies. GITHUB_ACTOR_ASSOCIATION (GitHub) or
		// DOCTOR_ACTOR_ROLE (platform-neutral, e.g. a GitLab access level).
		if applying {
			if role := doctorActorRole(); role != "" && !doctor.IsAuthorized(role) {
				return fmt.Errorf("actor role %q is not authorized to apply doctor fixes", role)
			}
		}

		_, files, err := collectDoctorDiff(opts.base)
		if err != nil {
			return err
		}

		root := doctorRepoRoot()
		paths, err := doctorPaths()
		if err != nil {
			return err
		}
		entries, err := listAllRecordEntries(paths.Root)
		if err != nil {
			return err
		}

		refs := recordRefsFromEntries(entries)
		result := doctor.Validate(refs, func(p string) bool {
			_, statErr := os.Stat(filepath.Join(root, p))
			return statErr == nil
		})
		for _, o := range result.Orphans {
			ui.Warning(os.Stderr, fmt.Sprintf("orphan: record %s points at missing file %q", o.Record, o.FileRef))
		}

		// Conflicts: changed files documented by more than one active record.
		conflicts := doctor.Conflicts(files, refs)

		// Apply phase: deprecate the chosen records, commit, re-check.
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

			remaining := doctor.RemainingConflicts(conflicts, deprecated)
			if len(remaining) > 0 || len(result.Orphans) > 0 {
				return fmt.Errorf("%d unresolved conflict(s) and %d orphan(s); merge blocked", len(remaining), len(result.Orphans))
			}
			ui.Success(os.Stderr, "doctor: all conflicts resolved.")
			return nil
		}

		// Detect phase: gate the merge on orphans or conflicts.
		if len(conflicts) > 0 {
			fmt.Fprintln(os.Stdout, doctor.RenderComment(conflicts))
		}
		if len(conflicts) > 0 || len(result.Orphans) > 0 {
			return fmt.Errorf("%d conflict(s) and %d orphan(s) block the merge", len(conflicts), len(result.Orphans))
		}

		ui.Success(os.Stderr, "doctor: records are consistent.")
		return nil
	}
}

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}
