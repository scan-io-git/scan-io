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
	"github.com/scan-io-git/scan-io/cmd/upload"
	"github.com/scan-io-git/scan-io/cmd/version"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"

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
		return handleExecuteError(err)
	}
	return 0
}

func handleExecuteError(err error) int {
	commandErr, ok := err.(*errors.CommandError)
	if !ok {
		commandErr = errors.NewCommandError(nil, nil, err, 1)
	}
	if config.IsCI(AppConfig) {
		if jsonErr := shared.PrintResultAsJSON(commandErr.Result); jsonErr != nil {
			Logger.Error("error serializing JSON result", "error", jsonErr)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", commandErr.Error())
	}
	return commandErr.ExitCode
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
	sarifissues.Init(AppConfig, Logger.Named("sarif-issues"))
	version.Init(AppConfig, Logger.Named("version"))
	tohtml.Init(AppConfig, Logger.Named("to-html"))
	upload.Init(AppConfig, Logger.Named("upload"))
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
	rootCmd.AddCommand(tohtml.ToHtmlCmd)
	rootCmd.AddCommand(upload.UploadCmd)
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}
