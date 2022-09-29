/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
	"github.com/spf13/cobra"
)

// pluginMap is the map of plugins we can dispense.
// var scannerPluginMap = map[string]plugin.Plugin{
// 	"scanner": &shared.ScannerPlugin{},
// }

func scanProject(scannerPluginName string, projects []string) {
	// Create an hclog.Logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin-scanner",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	// We're a host! Start by launching the plugin process.
	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	pluginsFolder := filepath.Join(home, "/.scanio/plugins")

	pluginPath := filepath.Join(pluginsFolder, scannerPluginName)
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         shared.PluginMap,
		Cmd:             exec.Command(pluginPath),
		Logger:          logger,
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		log.Fatal(err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("scanner")
	if err != nil {
		log.Fatal(err)
	}

	scanner := raw.(shared.Scanner)

	logger.Info("Scanner plugin initialized. Scan projects", "total", len(projects))

	for i, project := range projects {
		logger.Info("Run scan", "#", i+1, "project", project)
		ok := scanner.Scan(project)
		logger.Info("Scan finished", "#", i+1, "project", project, "statusOK", ok)
	}
	logger.Info(fmt.Sprintf("Completed scan of %d projects", len(projects)))
}

// analyseCmd represents the analyse command
var analyseCmd = &cobra.Command{
	Use:   "analyse",
	Short: "A brief description of your command",
	// 	Long: `A longer description that spans multiple lines and likely contains examples
	// and usage of using your command. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		err := cmd.Flags().Parse(args)
		if err != nil {
			panic("parse args error")
		}

		scannerPluginName, err := cmd.Flags().GetString("scanner")
		if err != nil {
			panic("get 'scanner' arg error")
		}

		projects, err := cmd.Flags().GetStringSlice("projects")
		if err != nil {
			panic(err)
		}
		if len(projects) == 0 {
			panic("specify at least one 'project' to scan")
		}

		scanProject(scannerPluginName, projects)
	},
}

func init() {
	rootCmd.AddCommand(analyseCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// analyseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// analyseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	analyseCmd.Flags().String("scanner", "semgrep", "scanner plugin name")
	analyseCmd.Flags().StringSlice("projects", []string{}, "Projects to scan")
}
