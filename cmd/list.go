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

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
	"github.com/spf13/cobra"
)

var (
	vcs         string
	vcsUrl      string
	outputFile  string
	maxProjects int
)

func do() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin-vcs",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	pluginsFolder := filepath.Join(home, "/.scanio/plugins")

	pluginPath := filepath.Join(pluginsFolder, vcs)
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         shared.PluginMap,
		Cmd:             exec.Command(pluginPath),
		Logger:          logger,
	})
	defer client.Kill()

	rpcClient, err := client.Client()
	if err != nil {
		log.Fatal(err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("vcs")
	if err != nil {
		log.Fatal(err)
	}

	vcs := raw.(shared.VCS)
	projects := vcs.ListProjects(shared.VCSListProjectsRequest{VCSURL: vcsUrl, MaxProjects: maxProjects})
	fmt.Printf("returned %d results\n", len(projects))

	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	datawriter := bufio.NewWriter(file)

	for _, data := range projects {
		_, _ = datawriter.WriteString(data + "\n")
	}

	datawriter.Flush()
	file.Close()

}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",
	// 	Long: `A longer description that spans multiple lines and likely contains examples
	// and usage of using your command. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Flags().Parse(args)
		if len(outputFile) == 0 {
			panic("'outputFile' must be specified")
		}
		do()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	listCmd.Flags().StringVar(&vcs, "vcs", "gitlab", "vcs plugin name")
	listCmd.Flags().StringVar(&vcsUrl, "vcs-url", "gitlab.com", "url to vcs")
	listCmd.Flags().StringVarP(&outputFile, "output", "f", "", "output file")
	listCmd.Flags().IntVar(&maxProjects, "max", 0, "max projects to list")
}
