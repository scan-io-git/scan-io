package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/cmd/analyse"
	"github.com/scan-io-git/scan-io/cmd/version"
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
	rootCmd.Flags().BoolP("help", "h", false, "Show help for Scanio.")
	rootCmd.AddCommand(analyse.AnalyseCmd)
	rootCmd.AddCommand(version.NewVersionCmd())
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	var err error
	AppConfig, err = config.LoadConfig(cfgFile)
	if err != nil {
		// TODO: Use a global logger
		fmt.Printf("Failed to load config file: %v\n", err)
		fmt.Println("Using default empty configuration")
	}

	if err := config.ValidateConfig(AppConfig); err != nil {
		fmt.Printf("Error validating config: %v\n", err)
		os.Exit(1)
	}

	analyse.Init(AppConfig)
	version.Init(AppConfig)
}
