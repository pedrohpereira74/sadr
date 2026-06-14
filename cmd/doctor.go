package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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

// collectDoctorDiff returns the raw diff against base and the list of changed files.
func collectDoctorDiff(base string) (string, []string, error) {
	diff, err := gitDiffFn(base)
	if err != nil {
		return "", nil, fmt.Errorf("git diff against %q failed: %w", base, err)
	}
	return diff, extractFilesFromDiff(diff), nil
}

func newDoctorCmd() *cobra.Command {
	opts := &doctorOptions{}

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Audit records against the diff (CI gatekeeper)",
		Long: "Validate records and detect API contract drift introduced by a pull request.\n" +
			"Designed to run in CI (--ci) as a merge gatekeeper.",
		Run: runDoctor(opts),
	}

	cmd.Flags().BoolVar(&opts.ci, "ci", false, "Non-interactive CI mode with structured output for ChatOps")
	cmd.Flags().StringVar(&opts.base, "base", "main", "Base branch of the pull request")
	cmd.Flags().StringVar(&opts.apply, "apply", "", "Comma-separated drift IDs approved for rewrite (triggers the apply phase)")
	return cmd
}

func runDoctor(opts *doctorOptions) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		ids := parseApplyIDs(opts.apply)
		phase := "detect"
		if len(ids) > 0 {
			phase = "apply"
		}
		ui.Info(os.Stderr, fmt.Sprintf("doctor: phase=%s base=%s ci=%v", phase, opts.base, opts.ci))
		if len(ids) > 0 {
			ui.Info(os.Stderr, fmt.Sprintf("doctor: approved drift ids: %s", strings.Join(ids, ", ")))
		}

		_, files, err := collectDoctorDiff(opts.base)
		if err != nil {
			ui.Error(os.Stderr, err.Error())
			return
		}
		ui.Info(os.Stderr, fmt.Sprintf("doctor: %d changed file(s) vs %s", len(files), opts.base))

		ui.Info(os.Stderr, "doctor: not implemented yet (scaffold).")
	}
}

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}
