package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/cmd/analyse"
	"github.com/scan-io-git/scan-io/cmd/fetch"
	"github.com/scan-io-git/scan-io/cmd/list"
	"github.com/scan-io-git/scan-io/cmd/version"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// Global variables for configuration and the command.
var (
	AppConfig *config.Config
	cfgFile   string
	rootCmd   = &cobra.Command{
		Use:                   "scanio [command]",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Short:                 "Comprehensive tool orchestration for security checks",
		Long: `Scanio is an orchestrator that consolidates various security scanning capabilities, including static code analysis, secret detection, dependency analysis, etc.

  Learn more at: https://github.com/scan-io-git/scan-io`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// initConfig reads the configuration file and initializes the commands with the loaded configuration.
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

	list.Init(AppConfig)
	fetch.Init(AppConfig)
	analyse.Init(AppConfig)
	version.Init(AppConfig)
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().BoolP("help", "h", false, "Show help for Scanio.")
	rootCmd.AddCommand(list.ListCmd)
	rootCmd.AddCommand(fetch.FetchCmd)
	rootCmd.AddCommand(analyse.AnalyseCmd)
	rootCmd.AddCommand(version.NewVersionCmd())
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .config.yml)")
}
