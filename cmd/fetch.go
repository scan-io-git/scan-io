/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/shared"
)

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
// var handshakeConfig = plugin.HandshakeConfig{
// 	ProtocolVersion:  1,
// 	MagicCookieKey:   "BASIC_PLUGIN",
// 	MagicCookieValue: "hello",
// }

// pluginMap is the map of plugins we can dispense.
// var vcsPluginMap = map[string]plugin.Plugin{
// 	"vcs": &shared.VCSPlugin{},
// }

func listProjects(vcsPluginName string, vcsUrl string, org string, authType string, sshKey string) []string {
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
	raw, err := rpcClient.Dispense("vcs")
	if err != nil {
		log.Fatal(err)
	}

	// We should have a Greeter now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	vcs := raw.(shared.VCS)

	res := vcs.ListProjects(shared.VCSListProjectsRequest{Organization: org, VCSURL: vcsUrl})

	logger.Info(fmt.Sprintf("'ListProjects' returned %d projects", len(res)))

	return res
}

func fetchProjects(vcsPluginName string, vcsUrl string, projects []string, threads int, authType string, sshKey string) {

	// We're a host! Start by launching the plugin process.
	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	pluginsFolder := filepath.Join(home, "/.scanio/plugins")
	pluginPath := filepath.Join(pluginsFolder, vcsPluginName)

	corelogger := hclog.New(&hclog.LoggerOptions{
		Name:   "core",
		Output: os.Stdout,
		// Level:  hclog.Info,
		Level: hclog.Debug,
	})
	corelogger.Info(fmt.Sprintf("Fetching %d projects in total, %d concurrent goroutines", len(projects), threads))

	maxGoroutines := threads
	guard := make(chan struct{}, maxGoroutines)
	var wg sync.WaitGroup
	for i, project := range projects {
		corelogger.Info("Begin fetch project", "#", i+1, "project", project)
		guard <- struct{}{} // would block if guard channel is already filled
		wg.Add(1)
		go func(i int, project string) {
			defer wg.Done()
			corelogger.Info("Goroutine started", "#", i+1, "project", project)

			// Create an hclog.Logger
			logger := hclog.New(&hclog.LoggerOptions{
				Name:   "plugin-vcs",
				Output: os.Stdout,
				// Level:  hclog.Info,
				Level: hclog.Debug,
			})

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
			raw, err := rpcClient.Dispense("vcs")
			if err != nil {
				log.Fatal(err)
			}

			// We should have a Greeter now! This feels like a normal interface
			// implementation but is in fact over an RPC connection.
			vcs := raw.(shared.VCS)

			logger.Info("Fetching...", "#", i+1, "project", project)
			args := shared.VCSFetchRequest{
				Project:  project,
				AuthType: authType,
				SSHKey:   sshKey,
				VCSURL:   vcsUrl,
			}
			res := vcs.Fetch(args)
			logger.Info("Fetching finished...", "#", i+1, "project", project, "res", res)

			<-guard
		}(i, project)
	}
	corelogger.Debug("Runned all goruotines, waiting for finishing them all")
	wg.Wait()
	corelogger.Debug("All goroutines are finished.")
}

func readProjectsFromFile(inputFile string) ([]string, error) {
	readFile, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	fileScanner := bufio.NewScanner(readFile)

	fileScanner.Split(bufio.ScanLines)

	projects := []string{}
	for fileScanner.Scan() {
		projects = append(projects, fileScanner.Text())
	}

	readFile.Close()
	return projects, nil
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
		inputFile, err := cmd.Flags().GetString("input-file")
		if err != nil {
			panic("get 'input-file' arg error")
		}
		org, err := cmd.Flags().GetString("org")
		if err != nil {
			panic("get 'org' arg error")
		}
		threads, err := cmd.Flags().GetInt("threads")
		if err != nil {
			panic("get 'threads' arg error")
		}
		authType, err := cmd.Flags().GetString("auth-type")
		if err != nil {
			panic("get 'auth-type' arg error")
		}
		sshKey, err := cmd.Flags().GetString("ssh-key")
		if err != nil {
			panic("get 'ssh-key' arg error")
		}
		vcsUrl, err := cmd.Flags().GetString("vcs-url")
		if err != nil {
			panic("get 'vcs-url' arg error")
		}

		if authType != "none" && authType != "ssh" {
			panic("unknown auth-type")
		}

		// if len() == "ssh" && len(sshKey) == 0 {
		// 	panic("specify ssh-key with auth-type 'ssh'")
		// }

		inputCount := 0
		if len(org) > 0 {
			inputCount += 1
		}
		if len(projects) > 0 {
			inputCount += 1
		}
		if len(inputFile) > 0 {
			inputCount += 1
		}
		if inputCount != 1 {
			panic("you must specify one of 'org', 'projects' or 'input-file")
		}
		if len(org) > 0 {
			projects = listProjects(vcsPluginName, vcsUrl, org, authType, sshKey)
		}

		if len(inputFile) > 0 {
			projectFromFile, err := readProjectsFromFile(inputFile)
			if err != nil {
				log.Fatal(err)
			}
			projects = projectFromFile
		}

		if len(projects) > 0 {
			fetchProjects(vcsPluginName, vcsUrl, projects, threads, authType, sshKey)
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
	fetchCmd.Flags().String("vcs-url", "gitlab.com", "vcs url")
	fetchCmd.Flags().StringSlice("projects", []string{}, "list of projects to fetch")
	fetchCmd.Flags().StringP("input-file", "f", "", "file with list of projects to fetch")
	fetchCmd.Flags().String("org", "", "fetch projects from this organization")
	fetchCmd.Flags().IntP("threads", "j", 1, "number of concurrent goroutines")
	fetchCmd.Flags().String("auth-type", "none", "Type of authentication: 'none' or 'ssh'")
	fetchCmd.Flags().String("ssh-key", "", "Path to ssh key")
}
