package cmd

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
)

type RunOptionsIntegrationVCS struct {
	VCSPlugName   string
	VCSURL        string
	Action        string
	Namespace     string
	Repository    string
	Login         string
	PullRequestId int
	Role          string
	Status        string
	Comment       string
}

type Arguments interface{}

var (
	allArgumentsIntegrationVCS RunOptionsIntegrationVCS
	resultIntegrationVCS       shared.GenericResult
	execExampleIntegrationVCS  = `  TODO # VCS plugin integrations for different actions
  scanio integration-vcs --vcs bitbucket --vcs-url example.com -o /Users/root/.scanio/output.file
  
  # Listing all repositories by a project in a VCS
  scanio list --vcs bitbucket --vcs-url example.com --namespace PROJECT -o /Users/root/.scanio/PROJECT.file`
)

var integrationVcsCmd = &cobra.Command{
	Use:                   "integration-vcs --vcs PLUGIN_NAME --output /local_path/output.file [--language LANGUAGE] (--vcs-url VCS_DOMAIN_NAME --namespace NAMESPACE | <url>)",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               execExampleIntegrationVCS,
	Short:                 "The command's function is VCS integrations for different actions",
	Long: `The command's function is VCS integrations for different actions

List of actions for bitbucket:
- Check a pull request existence and retrieve information
- Add reviewer to a pull request
  
List of actions for gitlab:
- nothing is implemented

List of actions for github:
- nothing is implemented`,

	RunE: func(cmd *cobra.Command, args []string) error {
		var arguments Arguments
		checkArgs := func() error {
			if err := validateCommonArguments(); err != nil {
				return err
			}
			switch allArgumentsIntegrationVCS.Action {
			case "checkPR":
				arguments = shared.VCSRetrivePRInformationRequest{
					VCSRequestBase: shared.VCSRequestBase{
						VCSURL:        allArgumentsIntegrationVCS.VCSURL,
						Action:        allArgumentsIntegrationVCS.Action,
						Namespace:     allArgumentsIntegrationVCS.Namespace,
						Repository:    allArgumentsIntegrationVCS.Repository,
						PullRequestId: allArgumentsIntegrationVCS.PullRequestId,
					},
				}
			case "addRoleToPR":
				if len(allArgumentsIntegrationVCS.Login) == 0 {
					return fmt.Errorf("The 'login' flag must be specified!")
				}
				if len(allArgumentsIntegrationVCS.Role) == 0 {
					return fmt.Errorf("The 'role' flag must be specified!")
				}
				arguments = shared.VCSAddRoleToPRRequest{
					VCSRequestBase: shared.VCSRequestBase{
						VCSURL:        allArgumentsIntegrationVCS.VCSURL,
						Action:        allArgumentsIntegrationVCS.Action,
						Namespace:     allArgumentsIntegrationVCS.Namespace,
						Repository:    allArgumentsIntegrationVCS.Repository,
						PullRequestId: allArgumentsIntegrationVCS.PullRequestId,
					},
					Login: allArgumentsIntegrationVCS.Login,
					Role:  allArgumentsIntegrationVCS.Role,
				}
			case "setStatusOfPR":
				if len(allArgumentsIntegrationVCS.Login) == 0 {
					return fmt.Errorf("The 'login' flag must be specified!")
				}
				if len(allArgumentsIntegrationVCS.Status) == 0 {
					return fmt.Errorf("The 'status' flag must be specified!")
				}
				arguments = shared.VCSSetStatusOfPRRequest{
					VCSRequestBase: shared.VCSRequestBase{
						VCSURL:        allArgumentsIntegrationVCS.VCSURL,
						Action:        allArgumentsIntegrationVCS.Action,
						Namespace:     allArgumentsIntegrationVCS.Namespace,
						Repository:    allArgumentsIntegrationVCS.Repository,
						PullRequestId: allArgumentsIntegrationVCS.PullRequestId,
					},
					Login:  allArgumentsIntegrationVCS.Login,
					Status: allArgumentsIntegrationVCS.Status,
				}
			case "addComment":
				if len(allArgumentsIntegrationVCS.Comment) == 0 {
					return fmt.Errorf("The 'comment' flag must be specified!")
				}
				arguments = shared.VCSAddCommentToPRRequest{
					VCSRequestBase: shared.VCSRequestBase{
						VCSURL:        allArgumentsIntegrationVCS.VCSURL,
						Action:        allArgumentsIntegrationVCS.Action,
						Namespace:     allArgumentsIntegrationVCS.Namespace,
						Repository:    allArgumentsIntegrationVCS.Repository,
						PullRequestId: allArgumentsIntegrationVCS.PullRequestId,
					},
					Comment: allArgumentsIntegrationVCS.Comment,
				}
			default:
				return fmt.Errorf("The action is not implemented %v", allArgumentsIntegrationVCS.Action)

			}

			logger := shared.NewLogger("core-integration-vcs")

			shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, allArgumentsIntegrationVCS.VCSPlugName, func(raw interface{}) {
				vcsName := raw.(shared.VCS)
				result, err := performAction(allArgumentsIntegrationVCS.Action, vcsName, arguments)

				if err != nil {
					resultIntegrationVCS = shared.GenericResult{Args: arguments, Result: result, Status: "FAILED", Message: err.Error()}
					logger.Error("A function of VCS integrations is failed", "action", allArgumentsIntegrationVCS.Action)
					logger.Error("Error", "message", resultIntegrationVCS.Message)
				} else {
					resultIntegrationVCS = shared.GenericResult{Args: arguments, Result: result, Status: "OK", Message: ""}
					logger.Info("A function of VCS integrations finished with", "status", resultIntegrationVCS.Status, "action", allArgumentsIntegrationVCS.Action)
				}
			})
			logger.Debug("Integration result", "result", resultIntegrationVCS)
			shared.WriteJsonFile(fmt.Sprintf("%v/VCS-integration.result", shared.GetScanioHome()), logger, resultIntegrationVCS)
			return nil
		}

		if err := checkArgs(); err != nil {
			return err
		}
		return nil
	},
}

func validateCommonArguments() error {
	if len(allArgumentsIntegrationVCS.VCSPlugName) == 0 {
		return fmt.Errorf("The 'vcs' flag must be specified!")
	}
	if len(allArgumentsIntegrationVCS.Action) == 0 {
		return fmt.Errorf("The 'action' flag must be specified!")
	}
	if len(allArgumentsIntegrationVCS.VCSURL) == 0 {
		return fmt.Errorf("The 'vcs-url' flag must be specified!")
	}
	if len(allArgumentsIntegrationVCS.Namespace) == 0 {
		return fmt.Errorf("The 'namespace' flag must be specified!")
	}
	if len(allArgumentsIntegrationVCS.Repository) == 0 {
		return fmt.Errorf("The 'repository' flag must be specified!")
	}
	if allArgumentsIntegrationVCS.PullRequestId == 0 {
		return fmt.Errorf("The 'pull-request-id' flag must be specified!")
	}
	return nil
}

func performAction(action string, vcsName shared.VCS, args Arguments) (interface{}, error) {
	switch action {
	case "checkPR":
		checkPRArgs, ok := args.(shared.VCSRetrivePRInformationRequest)
		if !ok {
			return nil, fmt.Errorf("Invalid argument type for action 'checkPR'")
		}
		return vcsName.RetrivePRInformation(checkPRArgs)
	case "addRoleToPR":
		addReviewArgs, ok := args.(shared.VCSAddRoleToPRRequest)
		if !ok {
			return nil, fmt.Errorf("Invalid argument type for action 'addRoleToPR'")
		}
		return vcsName.AddRoleToPR(addReviewArgs)
	case "setStatusOfPR":
		setStatusArgs, ok := args.(shared.VCSSetStatusOfPRRequest)
		if !ok {
			return nil, fmt.Errorf("Invalid argument type for action 'addRoleToPR'")
		}
		return vcsName.SetStatusOfPR(setStatusArgs)
	case "addComment":
		addComment, ok := args.(shared.VCSAddCommentToPRRequest)
		if !ok {
			return nil, fmt.Errorf("Invalid argument type for action 'addComment'")
		}
		return vcsName.AddComment(addComment)
	default:
		return nil, fmt.Errorf("Unsupported action: %s", action)
	}
}

func init() {
	rootCmd.AddCommand(integrationVcsCmd)

	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.VCSPlugName, "vcs", "", "the plugin name of the VCS used. Eg. bitbucket, gitlab, github, etc.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.VCSURL, "vcs-url", "", "URL to a root of the VCS API. Eg. github.com.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Action, "action", "", "the action to execute.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Namespace, "namespace", "", "the name of a specific namespace. Namespace for Gitlab is an organization, for Bitbucket_v1 is a project.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Repository, "repository", "", "the name of a specific repository.")
	integrationVcsCmd.Flags().IntVar(&allArgumentsIntegrationVCS.PullRequestId, "pull-request-id", 0, "the id of specific PR form the repository.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Login, "login", "", "login for integrations. For example, add reviewer with this login to PR.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Role, "role", "", "role for integrations. For example, add a person with specific role to PR.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Status, "status", "", "status for integrations. For example, set a status of PR.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Comment, "comment", "", "comment for integrations. The text will be used like a comment to PR")
}
