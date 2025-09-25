package createissue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// RunOptions holds flags for the create-issue command.
type RunOptions struct {
	Namespace  string `json:"namespace,omitempty"`
	Repository string `json:"repository,omitempty"`
	Title      string `json:"title,omitempty"`
	Body       string `json:"body,omitempty"`
}

var (
	AppConfig *config.Config
	opts      RunOptions

	// CreateIssueCmd represents the command to create a GitHub issue.
	CreateIssueCmd = &cobra.Command{
		Use:                   "create-issue --namespace NAMESPACE --repository REPO --title TITLE [--body BODY]",
		Short:                 "Create a GitHub issue (minimal command)",
		Example:               "go run ./main.go create-issue --namespace scan-io-git --repository scanio-test --title 'My Title' --body 'My Body'",
		Hidden:                true,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
				return cmd.Help()
			}

			if err := validate(&opts); err != nil {
				return errors.NewCommandError(opts, nil, err, 1)
			}

			lg := logger.NewLogger(AppConfig, "create-issue")

			// Build request for VCS plugin
			req := shared.VCSIssueCreationRequest{
				VCSRequestBase: shared.VCSRequestBase{
					RepoParam: shared.RepositoryParams{
						Namespace:  opts.Namespace,
						Repository: opts.Repository,
					},
					Action: "createIssue",
				},
				Title: opts.Title,
				Body:  opts.Body,
			}

			var createdIssueNumber int
			err := shared.WithPlugin(AppConfig, "plugin-vcs", shared.PluginTypeVCS, "github", func(raw interface{}) error {
				vcs, ok := raw.(shared.VCS)
				if !ok {
					return fmt.Errorf("invalid VCS plugin type")
				}
				num, err := vcs.CreateIssue(req)
				if err != nil {
					return err
				}
				createdIssueNumber = num
				return nil
			})
			if err != nil {
				lg.Error("failed to create issue via plugin", "error", err)
				return errors.NewCommandError(opts, nil, fmt.Errorf("create issue failed: %w", err), 2)
			}

			lg.Info("issue created", "number", createdIssueNumber)
			fmt.Printf("Created issue #%d\n", createdIssueNumber)
			return nil
		},
	}
)

// Init wires config into this command.
func Init(cfg *config.Config) { AppConfig = cfg }

func init() {
	CreateIssueCmd.Flags().StringVar(&opts.Namespace, "namespace", "", "GitHub org/user")
	CreateIssueCmd.Flags().StringVar(&opts.Repository, "repository", "", "Repository name")
	CreateIssueCmd.Flags().StringVar(&opts.Title, "title", "", "Issue title")
	CreateIssueCmd.Flags().StringVar(&opts.Body, "body", "", "Issue body")
	CreateIssueCmd.Flags().BoolP("help", "h", false, "Show help for create-issue command.")
}

func validate(o *RunOptions) error {
	if o.Namespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	if o.Repository == "" {
		return fmt.Errorf("--repository is required")
	}
	if o.Title == "" {
		return fmt.Errorf("--title is required")
	}
	return nil
}
