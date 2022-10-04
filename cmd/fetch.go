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
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/shared"
)

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

// pluginMap is the map of plugins we can dispense.
var vcsPluginMap = map[string]plugin.Plugin{
	"vcs": &shared.VCSPlugin{},
}

func listProjects(vcsPluginName string, org string) []string {
	// Create an hclog.Logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin-vcs",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	// We're a host! Start by launching the plugin process.
	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	pluginsFolder := filepath.Join(home, "/.scanio/plugins")

	//
	// useless check?
	//
	// if _, err := os.Stat(pluginsFolder); os.IsNotExist(err) {
	// 	logger.Info("pluginsFolder '%s' does not exists. Creating...", pluginsFolder)
	// 	if err := os.MkdirAll(pluginsFolder, os.ModePerm); err != nil {
	// 		panic(err)
	// 	}
	// }

	pluginPath := filepath.Join(pluginsFolder, vcsPluginName)
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         vcsPluginMap,
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
	raw, err := rpcClient.Dispense("vcs")
	if err != nil {
		log.Fatal(err)
	}

	// We should have a Greeter now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	vcs := raw.(shared.VCS)

	res := vcs.ListAllRepos(org)

	logger.Info(fmt.Sprintf("'ListAllRepos' returned %d projects", len(res)))

	return res
	// fmt.Println(res)
}

func fetchProjects(vcsPluginName string, projects []string, threads int) {

	// We're a host! Start by launching the plugin process.
	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	pluginsFolder := filepath.Join(home, "/.scanio/plugins")

	//
	// useless check?
	//
	// if _, err := os.Stat(pluginsFolder); os.IsNotExist(err) {
	// 	logger.Info("pluginsFolder '%s' does not exists. Creating...", pluginsFolder)
	// 	if err := os.MkdirAll(pluginsFolder, os.ModePerm); err != nil {
	// 		panic(err)
	// 	}
	// }

	pluginPath := filepath.Join(pluginsFolder, vcsPluginName)

	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "core",
		Output: os.Stdout,
		Level:  hclog.Info,
		// Level:  hclog.Debug,
	})
	logger.Info(fmt.Sprintf("Fetching %d projects in total, %d concurrent goroutines", len(projects), threads))

	maxGoroutines := threads
	guard := make(chan struct{}, maxGoroutines)
	for i, project := range projects {
		guard <- struct{}{} // would block if guard channel is already filled
		go func(i int, project string) {

			// Create an hclog.Logger
			logger := hclog.New(&hclog.LoggerOptions{
				Name:   "plugin-vcs",
				Output: os.Stdout,
				// Level:  hclog.Info,
				Level: hclog.Debug,
			})

			client := plugin.NewClient(&plugin.ClientConfig{
				HandshakeConfig: handshakeConfig,
				Plugins:         vcsPluginMap,
				Cmd:             exec.Command(pluginPath),
				Logger:          logger,
			})
			// defer client.Kill()

			// Connect via RPC
			rpcClient, err := client.Client()
			if err != nil {
				log.Fatal(err)
			}

			// Request the plugin
			raw, err := rpcClient.Dispense("vcs")
			if err != nil {
				log.Fatal(err)
			}

			// We should have a Greeter now! This feels like a normal interface
			// implementation but is in fact over an RPC connection.
			vcs := raw.(shared.VCS)

			logger.Info("Fetching...", "#", i+1, "project", project)
			res := vcs.Fetch(project)
			logger.Info("Fetching finished...", "#", i+1, "project", project, "res", res)

			client.Kill()
			<-guard
		}(i, project)
	}
}

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "A brief description of your command",
	// 	Long: `A longer description that spans multiple lines and likely contains examples
	// and usage of using your command. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("fetch called")
		err := cmd.Flags().Parse(args)
		if err != nil {
			panic("parse args error")
		}
		vcsPluginName, err := cmd.Flags().GetString("vcs")
		if err != nil {
			panic("get 'vcs' arg error")
		}
		projects, err := cmd.Flags().GetStringSlice("projects")
		if err != nil {
			panic("get 'projects' arg error")
		}
		org, err := cmd.Flags().GetString("org")
		if err != nil {
			panic("get 'org' arg error")
		}
		threads, err := cmd.Flags().GetInt("threads")
		if err != nil {
			panic("get 'threads' arg error")
		}

		if len(org) > 0 && len(projects) > 0 {
			panic("specify only one of 'org' or 'projects'")
		}

		if len(org) == 0 && len(projects) == 0 {
			panic("specify at least one project in 'projects' or 'org'")
		}

		if len(org) > 0 {
			projects = listProjects(vcsPluginName, org)
		}

		if len(projects) > 0 {
			fetchProjects(vcsPluginName, projects, threads)
		}
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fetchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	fetchCmd.Flags().String("vcs", "gitlab", "vcs plugin name")
	fetchCmd.Flags().StringSlice("projects", []string{}, "list of projects to fetch")
	fetchCmd.Flags().String("org", "", "fetch projects from this organization")
	fetchCmd.Flags().IntP("threads", "j", 1, "number of concurrent goroutines")
}
