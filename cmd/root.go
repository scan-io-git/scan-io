package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:                   "scanio [command]",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Short:                 "Scanio is an orchestrator for a variety of tools.",
	Long: `Scanio is an orchestrator that consolidates various security scanning capabilities, 
including SAST, dynamic application security testing DAST, secret search, and dependency analysis.
`,
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
