package cmd

import (
	"fmt"

	// "github.com/scan-io-git/scan-io/internal/vcs"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
)

type RunOptionsList struct {
	VCSPlugName string
	VCSURL      string
	OutputFile  string
	Namespace   string
}

var (
	limit            int
	allArgumentsList RunOptionsList
	resultVCS        shared.ListFuncResult
)

func do() {
	logger := shared.NewLogger("core")

	shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, allArgumentsList.VCSPlugName, func(raw interface{}) {
		vcsName := raw.(shared.VCS)
		args := shared.VCSListReposRequest{VCSURL: allArgumentsList.VCSURL, Namespace: allArgumentsList.Namespace}
		projects, err := vcsName.ListRepos(args)

		if err != nil {
			resultVCS = shared.ListFuncResult{Args: args, Result: projects, Status: "FAILED", Message: err.Error()}
			logger.Error("Failed", "error", resultVCS.Message)
		} else {
			resultVCS = shared.ListFuncResult{Args: args, Result: projects, Status: "OK", Message: ""}
			logger.Info("ListRepos fuctions is finished with status", "status", resultVCS.Status)
			logger.Info("Amount of repositories", "numbers", len(projects))
		}

		shared.WriteJsonFile(resultVCS, allArgumentsList.OutputFile, logger)
	})
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",

	RunE: func(cmd *cobra.Command, args []string) error {
		checkArgs := func() error {
			if len(allArgumentsList.VCSPlugName) == 0 {
				return fmt.Errorf(("'vcs' flag must be specified"))
			}

			if len(allArgumentsList.VCSURL) == 0 {
				return fmt.Errorf(("'vcs-url' flag must be specified"))
			}

			if len(allArgumentsList.OutputFile) == 0 {
				return fmt.Errorf(("'outputFile' flag must be specified"))
			}
			return nil
		}

		if err := checkArgs(); err != nil {
			return err
		}
		do()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&allArgumentsList.VCSPlugName, "vcs", "", "VCS plugin name")
	listCmd.Flags().StringVar(&allArgumentsList.VCSURL, "vcs-url", "", "url to VCS API root")
	listCmd.Flags().StringVarP(&allArgumentsList.OutputFile, "output", "f", "", "output file")
	listCmd.Flags().StringVar(&allArgumentsList.Namespace, "namespace", "", "list repos in a particular namespac. for Gitlab - organization, for Bitbucket_v1 - project")
}
