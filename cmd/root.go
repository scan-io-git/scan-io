package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

var (
	cfgFile   string
	AppConfig *config.Config
	rootCmd   = &cobra.Command{
		Use:                   "scanio [command]",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Short:                 "Scanio is an orchestrator for a variety of tools.",
		Long: `Scanio is an orchestrator that consolidates various security scanning capabilities, 
	including SAST, dynamic application security testing DAST, secret search, and dependency analysis.
	`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .config.yml)")
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	AppConfig, err := config.LoadConfig(cfgFile)
	if err != nil {
		// TODO: Use a global logger
		fmt.Printf("Failed to load config file: %v\n", err)
		fmt.Println("Using default empty configuration")
	}

	if err := config.ValidateConfig(AppConfig); err != nil {
		fmt.Printf("Error validating config: %v\n", err)
		os.Exit(1)
	}
}
