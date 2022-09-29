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
	"strings"
	"sync"

	"github.com/gitsight/go-vcsurl"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
	"github.com/spf13/cobra"
)

// pluginMap is the map of plugins we can dispense.
// var scannerPluginMap = map[string]plugin.Plugin{
// 	"scanner": &shared.ScannerPlugin{},
// }

func getVCSURLInfo(VCSURL string, project string) (*vcsurl.VCS, error) {
	if strings.Contains(project, ":") {
		return vcsurl.Parse(project)
	}

	return vcsurl.Parse(fmt.Sprintf("https://%s/%s", VCSURL, project))
}

func scanProject(scannerPluginName string, vcsUrl string, projects []string, threads int) {
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

	logger.Info("Scanner plugin initialized. Scan projects", "total", len(projects))

	maxGoroutines := threads
	guard := make(chan struct{}, maxGoroutines)
	var wg sync.WaitGroup
	for i, project := range projects {
		guard <- struct{}{} // would block if guard channel is already filled
		wg.Add(1)
		go func(i int, project string) {
			defer wg.Done()

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

			logger.Info("Run scan", "#", i+1, "project", project)
			info, err := getVCSURLInfo(vcsUrl, project)
			if err != nil {
				log.Fatal(err)
			}
			ok := scanner.Scan(info.ID)
			logger.Info("Scan finished", "#", i+1, "project", project, "statusOK", ok)

			<-guard
		}(i, project)
	}
	logger.Info(fmt.Sprintf("Completed scan of %d projects", len(projects)))
	logger.Debug("Runned all goruotines, waiting for finishing them all")
	wg.Wait()
	logger.Debug("All goroutines are finished.")
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
		vcsUrl, err := cmd.Flags().GetString("vcs-url")
		if err != nil {
			panic("get 'vcs-url' arg error")
		}
		threads, err := cmd.Flags().GetInt("threads")
		if err != nil {
			panic("get 'threads' arg error")
		}

		projects, err := cmd.Flags().GetStringSlice("projects")
		if err != nil {
			panic(err)
		}
		inputFile, err := cmd.Flags().GetString("input-file")
		if err != nil {
			panic("get 'input-file' arg error")
		}

		inputCount := 0
		if len(projects) > 0 {
			inputCount += 1
		}
		if len(inputFile) > 0 {
			inputCount += 1
		}
		if inputCount != 1 {
			panic("you must specify one of 'projects' or 'input-file")
		}

		if len(inputFile) > 0 {
			projectFromFile, err := readProjectsFromFile(inputFile)
			if err != nil {
				log.Fatal(err)
			}
			projects = projectFromFile
		}
		if len(projects) == 0 {
			panic("specify at least one 'project' to scan")
		}

		scanProject(scannerPluginName, vcsUrl, projects, threads)
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
	analyseCmd.Flags().String("vcs-url", "gitlab.com", "vcs url")
	analyseCmd.Flags().StringSlice("projects", []string{}, "Projects to scan")
	analyseCmd.Flags().StringP("input-file", "f", "", "file with list of projects to fetch")
	analyseCmd.Flags().IntP("threads", "j", 1, "number of concurrent goroutines")
}
