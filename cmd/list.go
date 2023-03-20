package cmd

import (
	"fmt"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
)

type RunOptionsList struct {
	VCSPlugName string
	VCSURL      string
	OutputFile  string
	Namespace   string
	Language    string
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
		args := shared.VCSListReposRequest{
			VCSURL:    allArgumentsList.VCSURL,
			Namespace: allArgumentsList.Namespace,
			Language:  allArgumentsList.Language,
		}
		projects, err := vcsName.ListRepos(args)
		logger.Info(args.Language)

		if err != nil {
			resultVCS = shared.ListFuncResult{Args: args, Result: projects, Status: "FAILED", Message: err.Error()}
			logger.Error("Failed", "error", resultVCS.Message)
		} else {
			resultVCS = shared.ListFuncResult{Args: args, Result: projects, Status: "OK", Message: ""}
			logger.Info("A ListRepos fuction is finished with status", "status", resultVCS.Status)
			logger.Info("Amount of repositories are", "numbers", len(projects))
		}

		shared.WriteJsonFile(resultVCS, allArgumentsList.OutputFile, logger)
	})
}

var listCmd = &cobra.Command{
	Use:          "list [flags] \n  scanio list [flags] [url]",
	SilenceUsage: true,
	Short:        "The command's function is to list repositories from a version control system.",

	RunE: func(cmd *cobra.Command, args []string) error {
		checkArgs := func() error {
			if len(args) >= 2 {
				return fmt.Errorf("Invalid argument(s) received!")
			}

			if len(allArgumentsList.VCSPlugName) == 0 {
				return fmt.Errorf(("'vcs' flag must be specified!"))
			}
			if len(args) == 1 {
				if len(allArgumentsList.VCSURL) != 0 || len(allArgumentsList.Namespace) != 0 {
					return fmt.Errorf(("You can't use a specific url with 'namespace' and 'vcs-url' arguments!"))
				}

				URL := args[0]
				hostname, namespace, _, _, _, err := shared.ExtractRepositoryInfoFromURL(URL, allArgumentsList.VCSPlugName)
				if err != nil {
					return err
				}
				allArgumentsList.VCSURL = hostname
				allArgumentsList.Namespace = namespace
			} else {
				if len(allArgumentsList.VCSURL) == 0 {
					return fmt.Errorf(("'vcs-url' flag must be specified!"))
				}
				if len(allArgumentsList.Namespace) == 0 {
					return fmt.Errorf(("'namespace' flag must be specified!"))
				}
			}

			if len(allArgumentsList.Language) != 0 && allArgumentsList.VCSPlugName != "gitlab" {
				return fmt.Errorf(("'language' is supported only for a gitlab plugin."))
			}

			if len(allArgumentsList.OutputFile) == 0 {
				return fmt.Errorf(("'outputFile' flag must be specified!"))
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

	listCmd.Flags().StringVar(&allArgumentsList.VCSPlugName, "vcs", "", "the plugin name of the VCS used. Eg. bitbucket, gitlab, github, etc.")
	listCmd.Flags().StringVar(&allArgumentsList.VCSURL, "vcs-url", "", "URL to a root of the VCS API. Eg. github.com.")
	listCmd.Flags().StringVarP(&allArgumentsList.OutputFile, "output", "f", "", "the path to an output file.")
	listCmd.Flags().StringVar(&allArgumentsList.Namespace, "namespace", "", "the name of a specific namespace. Namespace for Gitlab is an organization, for Bitbucket_v1 is a project.")
	listCmd.Flags().StringVarP(&allArgumentsList.Language, "language", "l", "", "collect only projects that have code on specified language. It works only for Giblab.")
}
