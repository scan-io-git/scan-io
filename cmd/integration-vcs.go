package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
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
	CommentFile   string
	AttachFiles   []string
}

type Arguments interface{}

var (
	allArgumentsIntegrationVCS RunOptionsIntegrationVCS

	execExampleIntegrationVCS = `# Checking the PR existence
  scanio integration-vcs --vcs bitbucket --action checkPR --vcs-url example.com --namespace PROJECT --repository REPOSITORY --pull-request-id ID

  # Add a user to the PR
  scanio integration-vcs --vcs bitbucket --action addRoleToPR --vcs-url example.com --namespace PROJECT --repository REPOSITORY --pull-request-id ID --login scanio-bot --role REVIWER

  # Change a status of the PR
  scanio integration-vcs --vcs bitbucket --action setStatusOfPR --vcs-url example.com --namespace PROJECT --repository REPOSITORY --pull-request-id ID --login scanio-bot --status UNAPPROVED

  # Leave a comment in the PR
  scanio integration-vcs --vcs bitbucket --action addComment --vcs-url example.com --namespace PROJECT --repository REPOSITORY --pull-request-id ID --comment "Test text"
  `
)

var integrationVcsCmd = &cobra.Command{
	Use:                   "integration-vcs --vcs PLUGIN_NAME --vcs-url VCS_DOMAIN_NAME --namespace NAMESPACE --repository REPOSITORY --pull-request-id ID ...",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               execExampleIntegrationVCS,
	Short:                 "The command's function is VCS integrations for different actions",
	Long: `The command's function is VCS integrations for different actions

List of actions for bitbucket:
- Check the PR existence and retrieve information about the PR
- Add a user to the PR
- Change a status of the PR
- Leave a comment in the PR
  
List of actions for gitlab:
- Not implemented

List of actions for github:
- Not implemented`,

	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			arguments             Arguments
			resultsIntegrationVCS shared.GenericLaunchesResult
		)

		checkArgs := func() error {
			if err := validateCommonArguments(); err != nil {
				return err
			}
			switch allArgumentsIntegrationVCS.Action {
			case "checkPR":
				arguments = shared.VCSRetrievePRInformationRequest{
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
					return fmt.Errorf("'login' flag must be specified")
				}
				if len(allArgumentsIntegrationVCS.Role) == 0 {
					return fmt.Errorf("'role' flag must be specified")
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
					return fmt.Errorf("'login' flag must be specified")
				}
				if len(allArgumentsIntegrationVCS.Status) == 0 {
					return fmt.Errorf("'status' flag must be specified")
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
				if allArgumentsIntegrationVCS.Comment == "" && allArgumentsIntegrationVCS.CommentFile == "" {
					return fmt.Errorf("either 'comment' or 'comment-file' flag must be specified")
				}
				if allArgumentsIntegrationVCS.Comment != "" && allArgumentsIntegrationVCS.CommentFile != "" {
					return fmt.Errorf("only one of 'comment' or 'comment-file' flag can be specified, not both")
				}

				var commentContent string
				if allArgumentsIntegrationVCS.CommentFile != "" {
					expandedPath, err := files.ExpandPath(allArgumentsIntegrationVCS.CommentFile)
					if err != nil {
						return fmt.Errorf("failed to expand path '%s': %w", allArgumentsIntegrationVCS.CommentFile, err)
					}

					if err := files.ValidatePath(expandedPath); err != nil {
						return fmt.Errorf("failed to validate path '%s': %w", expandedPath, err)
					}

					data, err := os.ReadFile(expandedPath)
					if err != nil {
						return fmt.Errorf("failed to read comment file: %v", err)
					}
					commentContent = string(data)
				} else {
					commentContent = allArgumentsIntegrationVCS.Comment
				}

				arguments = shared.VCSAddCommentToPRRequest{
					VCSRequestBase: shared.VCSRequestBase{
						VCSURL:        allArgumentsIntegrationVCS.VCSURL,
						Action:        allArgumentsIntegrationVCS.Action,
						Namespace:     allArgumentsIntegrationVCS.Namespace,
						Repository:    allArgumentsIntegrationVCS.Repository,
						PullRequestId: allArgumentsIntegrationVCS.PullRequestId,
					},
					Comment:   commentContent,
					FilePaths: allArgumentsIntegrationVCS.AttachFiles,
				}
			default:
				return fmt.Errorf("ation is not implemented %v", allArgumentsIntegrationVCS.Action)

			}

			logger := logger.NewLogger(AppConfig, "core-integration-vcs")
			resultIntegrationVCS, integrationErr := integrationAction(AppConfig, shared.PluginTypeVCS, allArgumentsIntegrationVCS.VCSPlugName, arguments)

			resultsIntegrationVCS.Launches = append(resultsIntegrationVCS.Launches, resultIntegrationVCS)
			_, err := json.Marshal(resultsIntegrationVCS)
			if err != nil {
				logger.Error("Error", "message", err)
				return err
			}

			logger.Debug("Integration result", "result", resultsIntegrationVCS)
			if err := shared.WriteGenericResult(AppConfig, logger, resultsIntegrationVCS, fmt.Sprintf("VCS-INTEGRATION-%v", strings.ToUpper(allArgumentsIntegrationVCS.Action))); err != nil {
				logger.Error("failed to write result", "error", err)
				return err
			}

			if integrationErr != nil {
				return fmt.Errorf("vcs plugin integration failed. Integration arguments: %v. Error: %w", arguments, integrationErr)
			}
			return nil
		}

		if err := checkArgs(); err != nil {
			return err
		}
		return nil
	},
}

// scanRepo executes the scanning of a repository using the specified plugin.
func integrationAction(cfg *config.Config, pluginType, pluginName string, arguments interface{}) (shared.GenericResult, error) {
	var result shared.GenericResult
	err := shared.WithPlugin(cfg, "plugin-vcs", pluginType, pluginName, func(raw interface{}) error {
		vcsName, ok := raw.(shared.VCS)
		if !ok {
			return fmt.Errorf("invalid plugin type")
		}
		var err error
		result, err := performAction(allArgumentsIntegrationVCS.Action, vcsName, arguments)
		if err != nil {
			result = shared.GenericResult{Args: arguments, Result: result, Status: "FAILED", Message: err.Error()}
			// logger.Error("A function of VCS integrations is failed", "action", allArgumentsIntegrationVCS.Action)
			// logger.Error("Error", "message", result.Message)
			return err
		}
		result = shared.GenericResult{Args: arguments, Result: result, Status: "OK", Message: ""}
		// logger.Info("A function of VCS integrations is successfully", "status", result.Status, "action", allArgumentsIntegrationVCS.Action)

		return nil
	})

	return result, err
}

func validateCommonArguments() error {
	if len(allArgumentsIntegrationVCS.VCSPlugName) == 0 {
		return fmt.Errorf("'vcs' flag must be specified")
	}
	if len(allArgumentsIntegrationVCS.Action) == 0 {
		return fmt.Errorf("'action' flag must be specified")
	}
	if len(allArgumentsIntegrationVCS.VCSURL) == 0 {
		return fmt.Errorf("'vcs-url' flag must be specified")
	}
	if len(allArgumentsIntegrationVCS.Namespace) == 0 {
		return fmt.Errorf("'namespace' flag must be specified")
	}
	if len(allArgumentsIntegrationVCS.Repository) == 0 {
		return fmt.Errorf("'repository' flag must be specified")
	}
	if allArgumentsIntegrationVCS.PullRequestId == 0 {
		return fmt.Errorf("'pull-request-id' flag must be specified")
	}
	return nil
}

func performAction(action string, vcsName shared.VCS, args Arguments) (interface{}, error) {
	switch action {
	case "checkPR":
		checkPRArgs, ok := args.(shared.VCSRetrievePRInformationRequest)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for action 'checkPR'")
		}
		return vcsName.RetrievePRInformation(checkPRArgs)
	case "addRoleToPR":
		addReviewArgs, ok := args.(shared.VCSAddRoleToPRRequest)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for action 'addRoleToPR'")
		}
		return vcsName.AddRoleToPR(addReviewArgs)
	case "setStatusOfPR":
		setStatusArgs, ok := args.(shared.VCSSetStatusOfPRRequest)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for action 'addRoleToPR'")
		}
		return vcsName.SetStatusOfPR(setStatusArgs)
	case "addComment":
		addComment, ok := args.(shared.VCSAddCommentToPRRequest)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for action 'addComment'")
		}
		return vcsName.AddCommentToPR(addComment)
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

func init() {
	rootCmd.AddCommand(integrationVcsCmd)

	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.VCSPlugName, "vcs", "", "the plugin name of the VCS used. Eg. bitbucket, gitlab, github, etc.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.VCSURL, "vcs-url", "", "URL to a root of the VCS API. Eg. github.com.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Action, "action", "", "the action to execute")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Namespace, "namespace", "", "the name of a specific namespace. Namespace for Gitlab is an organization, for Bitbucket_v1 is a project")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Repository, "repository", "", "the name of a specific repository")
	integrationVcsCmd.Flags().IntVar(&allArgumentsIntegrationVCS.PullRequestId, "pull-request-id", 0, "the id of specific PR form the repository")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Login, "login", "", "login for integrations. For example, add reviewer with this login to PR")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Role, "role", "", "role for integrations. For example, add a person with specific role to PR")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Status, "status", "", "status for integrations. For example, set a status of PR")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.Comment, "comment", "", "Comment for integrations. This text will be used as a comment on the pull request.")
	integrationVcsCmd.Flags().StringVar(&allArgumentsIntegrationVCS.CommentFile, "comment-file", "", "File containing the comment text for integrations. This text will be used as a comment on the pull request.")
	integrationVcsCmd.Flags().StringSliceVar(&allArgumentsIntegrationVCS.AttachFiles, "files", nil, "Comma-separated list of paths to files. These files will be uploaded and attached to the comment.")
}
