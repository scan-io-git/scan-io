package listissues

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// RunOptions holds flags for the list-issues command.
type RunOptions struct {
	Namespace  string `json:"namespace,omitempty"`
	Repository string `json:"repository,omitempty"`
	State      string `json:"state,omitempty"` // open|closed|all
}

var (
	AppConfig *config.Config
	opts      RunOptions

	// ListIssuesCmd represents the command to list GitHub issues.
	ListIssuesCmd = &cobra.Command{
		Use:                   "list-issues --namespace NAMESPACE --repository REPO [--state open|closed|all]",
		Short:                 "List GitHub issues (minimal command)",
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

			lg := logger.NewLogger(AppConfig, "list-issues")

			// Build request for VCS plugin
			req := shared.VCSListIssuesRequest{
				VCSRequestBase: shared.VCSRequestBase{
					RepoParam: shared.RepositoryParams{
						Namespace:  opts.Namespace,
						Repository: opts.Repository,
					},
					Action: "listIssues",
				},
				State: opts.State,
			}

			var issues []shared.IssueParams
			err := shared.WithPlugin(AppConfig, "plugin-vcs", shared.PluginTypeVCS, "github", func(raw interface{}) error {
				vcs, ok := raw.(shared.VCS)
				if !ok {
					return fmt.Errorf("invalid VCS plugin type")
				}
				list, err := vcs.ListIssues(req)
				if err != nil {
					return err
				}
				issues = list
				return nil
			})
			if err != nil {
				lg.Error("failed to list issues via plugin", "error", err)
				return errors.NewCommandError(opts, nil, fmt.Errorf("list issues failed: %w", err), 2)
			}

			// Sort by updated desc for nicer output
			sort.Slice(issues, func(i, j int) bool { return issues[i].UpdatedDate > issues[j].UpdatedDate })

			if len(issues) == 0 {
				fmt.Println("No issues found")
				return nil
			}

			// Print concise table
			fmt.Printf("#  %-8s  %-7s  %-18s  %s\n", "NUMBER", "STATE", "AUTHOR", "TITLE")
			for _, it := range issues {
				upd := time.Unix(it.UpdatedDate, 0).UTC().Format(time.RFC3339)
				_ = upd // keep for future verbose mode
				fmt.Printf("-  %-8d  %-7s  %-18s  %s\n", it.Number, it.State, it.Author.UserName, it.Title)
			}
			return nil
		},
	}
)

// Init wires config into this command.
func Init(cfg *config.Config) { AppConfig = cfg }

func init() {
	ListIssuesCmd.Flags().StringVar(&opts.Namespace, "namespace", "", "GitHub org/user")
	ListIssuesCmd.Flags().StringVar(&opts.Repository, "repository", "", "Repository name")
	ListIssuesCmd.Flags().StringVar(&opts.State, "state", "open", "Issue state filter: open|closed|all")
	ListIssuesCmd.Flags().BoolP("help", "h", false, "Show help for list-issues command.")
}

func validate(o *RunOptions) error {
	if o.Namespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	if o.Repository == "" {
		return fmt.Errorf("--repository is required")
	}
	switch o.State {
	case "", "open", "closed", "all":
		return nil
	default:
		return fmt.Errorf("--state must be one of: open, closed, all")
	}
}
