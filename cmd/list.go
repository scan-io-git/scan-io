/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/scan-io-git/scan-io/libs/vcs"
	"github.com/scan-io-git/scan-io/shared"
	"github.com/spf13/cobra"
)

var (
	vcsName, vcsUrl, outputFile, namespace string
	limit                                  int
	resultVCS                              vcs.ListFuncResult
)

func do() {
	logger := shared.NewLogger("core")

	shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, vcsName, func(raw interface{}) {
		vcsName := raw.(vcs.VCS)
		args := vcs.VCSListReposRequest{VCSURL: vcsUrl, Limit: limit, Namespace: namespace}
		projects, err := vcsName.ListRepos(args)

		if err != nil {
			resultVCS = vcs.ListFuncResult{Args: args, Result: projects, Status: "FAILED", Message: err.Error()}
			logger.Error("Failed", "error", resultVCS.Message)
		} else {
			resultVCS = vcs.ListFuncResult{Args: args, Result: projects, Status: "OK", Message: ""}
			logger.Info("ListRepos fuctions is finished with status", resultVCS.Status)
			logger.Info("Amount of repositories", len(projects))
		}

		vcs.WriteJsonFile(resultVCS, outputFile, logger)
	})
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",

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

	listCmd.Flags().StringVar(&vcsName, "vcs", "", "VCS plugin name")
	listCmd.Flags().StringVar(&vcsUrl, "vcs-url", "", "url to VCS API root")
	listCmd.Flags().StringVarP(&outputFile, "output", "f", "", "output file")
	listCmd.Flags().StringVar(&namespace, "namespace", "", "list repos in a particular namespac. for Gitlab - organization, for Bitbucket_v1 - project")
	listCmd.Flags().IntVar(&limit, "limit", 0, "max projects to list")
}
