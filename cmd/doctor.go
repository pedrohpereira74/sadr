package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pedrohpereira74/sadr/internal/compress"
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

// parseApplyIDs splits the --apply CSV into trimmed, non-empty drift IDs.
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

// recordDocsForTargets gathers the documentation (sections) of the records named
// by the audit targets, de-duplicated, in target order.
func recordDocsForTargets(targets []doctor.AuditTarget, entries []storage.RecordEntry) []doctor.RecordDoc {
	byID := indexEntries(entries)
	var docs []doctor.RecordDoc
	seen := map[string]bool{}
	for _, tgt := range targets {
		for _, rid := range tgt.Records {
			if seen[rid] {
				continue
			}
			seen[rid] = true
			if e, ok := byID[rid]; ok {
				docs = append(docs, doctor.RecordDoc{
					ID:       rid,
					FileRef:  e.Record.FileRef,
					Sections: e.Record.Fields,
				})
			}
		}
	}
	return docs
}

// rewriteRequestsForDrifts builds rewrite requests from approved drifts, pulling
// each section's current content from its record.
func rewriteRequestsForDrifts(drifts []doctor.Drift, entries []storage.RecordEntry) []doctor.RewriteRequest {
	byID := indexEntries(entries)
	var reqs []doctor.RewriteRequest
	for _, d := range drifts {
		e, ok := byID[d.Record]
		if !ok {
			continue
		}
		reqs = append(reqs, doctor.RewriteRequest{
			Record:  d.Record,
			Section: d.Section,
			Current: e.Record.Fields[d.Section],
			Summary: d.Summary,
		})
	}
	return reqs
}

// applyRewrites writes each rewritten section back into its record file in place,
// returning the changed file paths (sorted) and the set of resolved drift IDs.
func applyRewrites(rewrites []doctor.Rewrite, entries []storage.RecordEntry) ([]string, map[string]bool, error) {
	byID := indexEntries(entries)
	s := storage.NewStorage("")
	changed := map[string]bool{}
	resolved := map[string]bool{}

	for _, rw := range rewrites {
		e, ok := byID[rw.Record]
		if !ok {
			continue
		}
		section := strings.ReplaceAll(strings.ToLower(rw.Section), " ", "_")
		e.Record.Fields[section] = rw.Content
		if err := s.UpdateRecord(e.Path, e.Record); err != nil {
			return nil, nil, fmt.Errorf("failed to update %s: %w", e.Path, err)
		}
		changed[e.Path] = true
		resolved[doctor.DriftID(rw.Record, section)] = true
	}

	paths := make([]string, 0, len(changed))
	for p := range changed {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, resolved, nil
}

// buildSkeletons reads each changed file under root and returns path->skeleton.
// Files that cannot be read (deleted in the diff, binary, outside root) are skipped.
func buildSkeletons(root string, files []string) map[string]string {
	skeletons := map[string]string{}
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue
		}
		skeletons[f] = doctor.Skeleton(string(data))
	}
	return skeletons
}

func newDoctorCmd() *cobra.Command {
	opts := &doctorOptions{}

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Audit records against the diff (CI gatekeeper)",
		Long: "Validate records and detect API contract drift introduced by a pull request.\n" +
			"Designed to run in CI (--ci) as a merge gatekeeper.",
		RunE:          runDoctor(opts),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolVar(&opts.ci, "ci", false, "Non-interactive CI mode with structured output for ChatOps")
	cmd.Flags().StringVar(&opts.base, "base", "main", "Base branch of the pull request")
	cmd.Flags().StringVar(&opts.apply, "apply", "", "Comma-separated drift IDs approved for rewrite (triggers the apply phase)")
	return cmd
}

func runDoctor(opts *doctorOptions) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		applyAll := strings.TrimSpace(opts.apply) == "all"
		ids := parseApplyIDs(opts.apply)
		applying := applyAll || len(ids) > 0

		// Defense in depth: the CI job is the first authorization gate (GitHub
		// issue_comment / GitLab manual job), but if it passes an actor role,
		// refuse unauthorized applies. GITHUB_ACTOR_ASSOCIATION (GitHub) or
		// DOCTOR_ACTOR_ROLE (platform-neutral, e.g. a GitLab access level).
		if applying {
			if role := doctorActorRole(); role != "" && !doctor.IsAuthorized(role) {
				return fmt.Errorf("actor role %q is not authorized to apply doctor fixes", role)
			}
		}

		diff, files, err := collectDoctorDiff(opts.base)
		if err != nil {
			return err
		}

		root := doctorRepoRoot()
		compressedDiff := compress.ZipSnippet(diff)
		skeletons := buildSkeletons(root, files)

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
		for _, c := range result.Collisions {
			ui.Warning(os.Stderr, fmt.Sprintf("collision: file %q referenced by %s", c.FileRef, strings.Join(c.Records, ", ")))
		}

		var drifts []doctor.Drift
		if targets := doctor.FilterChangedFiles(files, refs); len(targets) > 0 {
			docs := recordDocsForTargets(targets, entries)
			provider, apiKey, model := loadAIConfig()
			prompt := doctor.BuildDriftPrompt(compressedDiff, skeletons, docs)
			resp, gErr := generateTextFn(context.Background(), provider, prompt, apiKey, model, 0)
			if gErr != nil {
				return fmt.Errorf("drift detection failed: %w", gErr)
			}
			if drifts, err = doctor.ParseDrifts(resp); err != nil {
				return err
			}
		}

		// Apply phase: rewrite the approved drifts (commit lands in F8).
		if applying {
			approved := doctor.SelectDrifts(drifts, ids, applyAll)
			if len(approved) == 0 {
				ui.Info(os.Stderr, "doctor: no matching drifts to apply.")
				return nil
			}
			reqs := rewriteRequestsForDrifts(approved, entries)
			provider, apiKey, model := loadAIConfig()
			resp, gErr := generateTextFn(context.Background(), provider, doctor.BuildRewritePrompt(compressedDiff, reqs), apiKey, model, 0)
			if gErr != nil {
				return fmt.Errorf("rewrite failed: %w", gErr)
			}
			rewrites, rErr := doctor.ParseRewrites(resp)
			if rErr != nil {
				return rErr
			}

			changedPaths, resolved, aErr := applyRewrites(rewrites, entries)
			if aErr != nil {
				return aErr
			}
			if len(changedPaths) > 0 {
				msg := fmt.Sprintf("docs(sadr): doctor rewrote %d drifted section(s)", len(changedPaths))
				if cErr := gitCommitFn(root, changedPaths, msg); cErr != nil {
					return cErr
				}
				ui.Success(os.Stderr, fmt.Sprintf("doctor: rewrote and committed %d record(s)", len(changedPaths)))
			}

			// Any detected drift that was not resolved keeps the merge blocked.
			var unresolved int
			for _, d := range drifts {
				if !resolved[d.ID] {
					unresolved++
				}
			}
			if unresolved > 0 {
				return fmt.Errorf("%d drift(s) still unresolved; merge blocked", unresolved)
			}
			ui.Success(os.Stderr, "doctor: all detected drift resolved.")
			return nil
		}

		// Detect phase: gate the merge on validation failures or detected drift.
		if !result.OK() {
			return fmt.Errorf("record validation failed: %d orphan(s), %d collision(s)", len(result.Orphans), len(result.Collisions))
		}
		if len(drifts) == 0 {
			ui.Success(os.Stderr, "doctor: no contract drift detected.")
			return nil
		}

		// Emit the ChatOps comment (consumed by the workflow) and fail the check.
		fmt.Fprintln(os.Stdout, doctor.RenderComment(drifts))
		return fmt.Errorf("%d documentation drift(s) detected; reply /doctor apply to resolve", len(drifts))
	}
}

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}
