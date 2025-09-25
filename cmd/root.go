package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/cmd/analyse"
	"github.com/scan-io-git/scan-io/cmd/fetch"
	integrationvcs "github.com/scan-io-git/scan-io/cmd/integration-vcs"
	"github.com/scan-io-git/scan-io/cmd/list"
	sarifissues "github.com/scan-io-git/scan-io/cmd/sarif-issues"
	"github.com/scan-io-git/scan-io/cmd/version"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// Global variables for configuration and the command.
var (
	AppConfig *config.Config
	Logger    hclog.Logger
	cfgFile   string
	rootCmd   = &cobra.Command{
		Use:                   "scanio [command]",
		SilenceUsage:          true,
		SilenceErrors:         true,
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
		if commandErr, ok := err.(*errors.CommandError); ok {
			if config.IsCI(AppConfig) {
				shared.PrintResultAsJSON(Logger, commandErr.Result)
			} else {
				fmt.Printf("Error: %v\n", err.Error())
			}
			os.Exit(commandErr.ExitCode)
		}
		os.Exit(1)
	}
}

// initConfig reads the configuration file and initializes the commands with the loaded configuration.
func initConfig() {
	var err error
	AppConfig, err = config.LoadConfig(cfgFile)
	Logger = logger.NewLogger(AppConfig, "core")
	if err != nil {
		Logger.Warn("failed to load config file", "error", err)
		Logger.Warn("using default empty configuration")
	}

	if err := config.ValidateConfig(AppConfig); err != nil {
		Logger.Error("failed to validate Scanio config", "error", err)
		os.Exit(1)
	}

	list.Init(AppConfig)
	fetch.Init(AppConfig)
	analyse.Init(AppConfig)
	integrationvcs.Init(AppConfig)
	sarifissues.Init(AppConfig)
	version.Init(AppConfig)
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().BoolP("help", "h", false, "Show help for Scanio.")
	rootCmd.AddCommand(list.ListCmd)
	rootCmd.AddCommand(fetch.FetchCmd)
	rootCmd.AddCommand(analyse.AnalyseCmd)
	rootCmd.AddCommand(integrationvcs.IntegrationVCSCmd)
	rootCmd.AddCommand(sarifissues.SarifIssuesCmd)
	rootCmd.AddCommand(version.NewVersionCmd())
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}
