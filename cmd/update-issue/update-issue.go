package updateissue

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// RunOptions holds flags for the update-issue command.
type RunOptions struct {
	Namespace  string `json:"namespace,omitempty"`
	Repository string `json:"repository,omitempty"`
	Number     int    `json:"number,omitempty"`
	Title      string `json:"title,omitempty"`
	Body       string `json:"body,omitempty"`
	State      string `json:"state,omitempty"`
}

var (
	AppConfig *config.Config
	opts      RunOptions

	// UpdateIssueCmd represents the command to update a GitHub issue.
	UpdateIssueCmd = &cobra.Command{
		Use:                   "update-issue --namespace NAMESPACE --repository REPO --number N [--title TITLE] [--body BODY] [--state STATE]",
		Short:                 "Update a GitHub issue (title/body/state)",
		Example:               "scanio update-issue --namespace scan-io-git --repository scanio-test --number 4 --state closed",
		SilenceUsage:          true,
		Hidden:                true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
				return cmd.Help()
			}

			if err := validate(&opts); err != nil {
				return errors.NewCommandError(opts, nil, err, 1)
			}

			lg := logger.NewLogger(AppConfig, "update-issue")

			// Build request for VCS plugin
			req := shared.VCSIssueUpdateRequest{
				VCSRequestBase: shared.VCSRequestBase{
					RepoParam: shared.RepositoryParams{
						Namespace:  opts.Namespace,
						Repository: opts.Repository,
					},
					Action: "updateIssue",
				},
				Number: opts.Number,
				Title:  opts.Title,
				Body:   opts.Body,
				State:  opts.State,
			}

			var success bool
			err := shared.WithPlugin(AppConfig, "plugin-vcs", shared.PluginTypeVCS, "github", func(raw interface{}) error {
				vcs, ok := raw.(shared.VCS)
				if !ok {
					return fmt.Errorf("invalid VCS plugin type")
				}
				okResp, err := vcs.UpdateIssue(req)
				if err != nil {
					return err
				}
				success = okResp
				return nil
			})
			if err != nil {
				lg.Error("failed to update issue via plugin", "error", err)
				return errors.NewCommandError(opts, nil, fmt.Errorf("update issue failed: %w", err), 2)
			}

			if success {
				lg.Info("issue updated", "number", opts.Number)
				fmt.Printf("Updated issue #%d\n", opts.Number)
			} else {
				lg.Warn("issue update returned false", "number", opts.Number)
				fmt.Printf("Issue not updated (no-op?) #%d\n", opts.Number)
			}
			return nil
		},
	}
)

// Init wires config into this command.
func Init(cfg *config.Config) { AppConfig = cfg }

func init() {
	UpdateIssueCmd.Flags().StringVar(&opts.Namespace, "namespace", "", "GitHub org/user")
	UpdateIssueCmd.Flags().StringVar(&opts.Repository, "repository", "", "Repository name")
	UpdateIssueCmd.Flags().IntVar(&opts.Number, "number", 0, "Issue number")
	UpdateIssueCmd.Flags().StringVar(&opts.Title, "title", "", "New issue title")
	UpdateIssueCmd.Flags().StringVar(&opts.Body, "body", "", "New issue body")
	UpdateIssueCmd.Flags().StringVar(&opts.State, "state", "", "New issue state: open or closed")
	UpdateIssueCmd.Flags().BoolP("help", "h", false, "Show help for update-issue command.")
}

func validate(o *RunOptions) error {
	if o.Namespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	if o.Repository == "" {
		return fmt.Errorf("--repository is required")
	}
	if o.Number <= 0 {
		return fmt.Errorf("--number is required and must be > 0")
	}
	// at least one field to update must be provided
	if strings.TrimSpace(o.Title) == "" && strings.TrimSpace(o.Body) == "" && strings.TrimSpace(o.State) == "" {
		return fmt.Errorf("provide at least one of --title, --body, or --state")
	}
	if s := strings.ToLower(strings.TrimSpace(o.State)); s != "" && s != "open" && s != "closed" {
		return fmt.Errorf("--state must be 'open' or 'closed' if provided")
	}
	return nil
}
