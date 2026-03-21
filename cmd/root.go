package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sadr",
	Short: ":( sadr — snippets + architecture decision records",
	Long:  "Capture code with context. Because snippets without a \"why\" are sadr.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
