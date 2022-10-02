/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"log"
	"os"

	"github.com/scan-io-git/scan-io/shared"
	"github.com/spf13/cobra"
)

var (
	vcs        string
	vcsUrl     string
	outputFile string
	limit      int
)

func do() {

	logger := shared.NewLogger("core")

	shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, vcs, func(raw interface{}) {
		vcs := raw.(shared.VCS)
		projects := vcs.ListRepos(shared.VCSListReposRequest{VCSURL: vcsUrl, Limit: limit})
		logger.Info("ListRepos finished", "total", len(projects))

		file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("failed creating file: %s", err)
		}
		defer file.Close()

		datawriter := bufio.NewWriter(file)
		defer datawriter.Flush()

		for _, data := range projects {
			_, _ = datawriter.WriteString(data + "\n")
		}

		logger.Info("Results saved to file", "filepath", outputFile)
	})
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
	listCmd.Flags().IntVar(&limit, "limit", 0, "max projects to list")
}
