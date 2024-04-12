package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

type RunOptionsList struct {
	VCSPlugName string
	VCSURL      string
	OutputFile  string
	Namespace   string
	Repository  string
	Language    string
}

var (
	limit            int
	allArgumentsList RunOptionsList
	resultVCS        shared.GenericResult
	execExampleList  = `  # Listing all repositories in a VCS
  scanio list --vcs bitbucket --vcs-url example.com -o /Users/root/.scanio/output.file
  
  # Listing all repositories by a project in a VCS
  scanio list --vcs bitbucket --vcs-url example.com --namespace PROJECT -o /Users/root/.scanio/PROJECT.file

  # Listing all repositories in a VCS using URL
  scanio list --vcs bitbucket -o /Users/root/.scanio/PROJECT.file https://example.com/

  # Listing all repositories by a project using URL
  scanio list --vcs bitbucket -o /Users/root/.scanio/PROJECT.file https://example.com/projects/PROJECT/`
)

func do() {
	logger := logger.NewLogger(AppConfig, "core-list")

	shared.WithPlugin(AppConfig, "plugin-vcs", shared.PluginTypeVCS, allArgumentsList.VCSPlugName, func(raw interface{}) error {
		vcsName := raw.(shared.VCS)
		args := shared.VCSListReposRequest{
			VCSURL:    allArgumentsList.VCSURL,
			Namespace: allArgumentsList.Namespace,
			// Repository: allArgumentsList.Repository,
			Language: allArgumentsList.Language,
		}
		if len(allArgumentsList.Repository) != 0 {
			logger.Warn("Listing a particular repository is not supported. The namespace will be listed instead", "namespace", args.Namespace)
		}

		projects, err := vcsName.ListRepos(args)

		if err != nil {
			resultVCS = shared.GenericResult{Args: args, Result: projects, Status: "FAILED", Message: err.Error()}
			logger.Error("A function of listing repositories is failed")
		} else {
			resultVCS = shared.GenericResult{Args: args, Result: projects, Status: "OK", Message: ""}

			logger.Info("A function of listing repositories finished with", "status", resultVCS.Status)
			logger.Info("The amount of repositories is", "numbers", len(projects))
		}

		shared.WriteJsonFile(allArgumentsList.OutputFile, logger, resultVCS)
		return nil
	})
}

var listCmd = &cobra.Command{
	Use:                   "list --vcs PLUGIN_NAME --output /local_path/output.file [--language LANGUAGE] (--vcs-url VCS_DOMAIN_NAME --namespace NAMESPACE | <url>)",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               execExampleList,
	Short:                 "The command's function is to list repositories from a version control system",
	Long: `The command's function is to list repositories from a version control system

List of plugins:
  - bitbucket
  - gitlab
  - github`,

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
				hostname, namespace, repository, _, _, _, err := shared.ExtractRepositoryInfoFromURL(URL, allArgumentsList.VCSPlugName)
				if err != nil {
					return err
				}
				allArgumentsList.VCSURL = hostname
				allArgumentsList.Namespace = namespace
				allArgumentsList.Repository = repository
			} else {
				if len(allArgumentsList.VCSURL) == 0 {
					return fmt.Errorf(("'vcs-url' flag must be specified!"))
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
	listCmd.Flags().StringVarP(&allArgumentsList.OutputFile, "output", "o", "", "the path to an output file.")
	listCmd.Flags().StringVar(&allArgumentsList.Namespace, "namespace", "", "the name of a specific namespace. Namespace for Gitlab is an organization, for Bitbucket_v1 is a project.")
	listCmd.Flags().StringVarP(&allArgumentsList.Language, "language", "l", "", "collect only projects that have code on specified language. It's supported only for Giblab.")
}
