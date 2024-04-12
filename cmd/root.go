package cmd

import (
	"fmt"
	"os"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/spf13/cobra"
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

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .config.yml)")
}

func initConfig() {
	var err error

	if len(cfgFile) == 0 {
		cfgFile = "config.yml"
	}

	AppConfig, err = config.NewConfig(cfgFile)
	if err != nil {
		fmt.Println("initializing config file function is crashed - %v", err)
		os.Exit(1)
	}
}
