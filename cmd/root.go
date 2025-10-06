package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/cmd/analyse"
	"github.com/scan-io-git/scan-io/cmd/fetch"
	"github.com/scan-io-git/scan-io/cmd/list"
	"github.com/scan-io-git/scan-io/cmd/upload"
	"github.com/scan-io-git/scan-io/cmd/version"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"

	integrationvcs "github.com/scan-io-git/scan-io/cmd/integration-vcs"
	sarifcomments "github.com/scan-io-git/scan-io/cmd/sarif-comments"
	sarifissues "github.com/scan-io-git/scan-io/cmd/sarif-issues"
	tohtml "github.com/scan-io-git/scan-io/cmd/to-html"
)

// Global variables for configuration and the command.
var (
	AppConfig  *config.Config
	Logger     hclog.Logger
	closeLogFn logger.Close = func() error { return nil }
	cfgFile    string
	rootCmd    = &cobra.Command{
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
func Execute() int {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	defer func() {
		if closeLogFn != nil {
			_ = closeLogFn()
		}
	}()
	if err := rootCmd.Execute(); err != nil {
		if commandErr, ok := err.(*errors.CommandError); ok {
			if config.IsCI(AppConfig) {
				if err := shared.PrintResultAsJSON(commandErr.Result); err != nil {
					Logger.Error("error serializing JSON result", "error", err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err.Error())
			}
			return commandErr.ExitCode
		}
		return 1
	}
	return 0
}

// initConfig reads the configuration file and initializes the commands with the loaded configuration.
func initConfig() {
	var err error
	AppConfig, err = config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config file: %v\n", err)
		fmt.Fprintf(os.Stderr, "using default empty configuration")
	}

	if err := config.ValidateConfig(AppConfig); err != nil {
		fmt.Fprintf(os.Stderr, "failed to validate Scanio config: %v\n", err)
		os.Exit(1)
	}

	var logErr error
	Logger, closeLogFn, logErr = logger.NewLogger(AppConfig, "core")
	if logErr != nil {
		Logger.Warn("file logging disabled", "err", logErr)
	}

	list.Init(AppConfig, Logger.Named("list"))
	fetch.Init(AppConfig, Logger.Named("fetch"))
	analyse.Init(AppConfig, Logger.Named("analyse"))
	integrationvcs.Init(AppConfig, Logger.Named("integration-vcs"))
	version.Init(AppConfig, Logger.Named("version"))
	tohtml.Init(AppConfig, Logger.Named("to-html"))
	upload.Init(AppConfig, Logger.Named("upload"))
	sarifissues.Init(AppConfig, Logger.Named("sarif-issues"))
	sarifcomments.Init(AppConfig, Logger.Named("sarif-comments"))
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().BoolP("help", "h", false, "Show help for Scanio.")
	rootCmd.AddCommand(list.ListCmd)
	rootCmd.AddCommand(fetch.FetchCmd)
	rootCmd.AddCommand(analyse.AnalyseCmd)
	rootCmd.AddCommand(integrationvcs.IntegrationVCSCmd)
	rootCmd.AddCommand(sarifissues.SarifIssuesCmd)
	rootCmd.AddCommand(sarifcomments.SarifCommentsCmd)
	rootCmd.AddCommand(version.NewVersionCmd())
	rootCmd.AddCommand(tohtml.ToHtmlCmd)
	rootCmd.AddCommand(upload.UploadCmd)
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}
