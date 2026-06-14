package cmd

import (
	"fmt"
	"os"
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
		ui.Info(os.Stderr, "doctor: not implemented yet (scaffold).")
	}
}

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}
