package vcsintegrator

import (
	"fmt"

	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

const (
	VCSListing       = "list"
	VCSCheckPR       = "checkPR"
	VCSCommentPR     = "addComment"
	VCSAddRoleToPR   = "addRoleToPR"
	VCSSetStatusOfPR = "setStatusOfPR"
)

// VCSIntegrator represents the configuration and behavior of VCS integration actions.
type VCSIntegrator struct {
	PluginName string       // Name of the VCS plugin to use
	Action     string       // Action to perform
	logger     hclog.Logger // Logger for logging messages and errors
}

// RunOptionsIntegrationVCS holds the arguments for VCS integration actions.
type RunOptionsIntegrationVCS struct {
	VCSPluginName  string   `json:"vcs_plugin_name,omitempty"`
	Domain         string   `json:"domain,omitempty"`
	Namespace      string   `json:"namespace,omitempty"`
	Repository     string   `json:"repository,omitempty"`
	PullRequestID  string   `json:"pull_request_id,omitempty"`
	Action         string   `json:"action,omitempty"`
	Login          string   `json:"login,omitempty"`
	Language       string   `json:"language,omitempty"`
	OutputPath     string   `json:"output_path,omitempty"`
	Role           string   `json:"role,omitempty"`
	Status         string   `json:"status,omitempty"`
	Comment        string   `json:"comment,omitempty"`
	CommentFile    string   `json:"comment_file,omitempty"`
	AttachFiles    []string `json:"attach_files,omitempty"`
	LocalTipCommit string   `json:"local_tip_commit,omitempty"` // Tip commit hash to verify is local and remote tip commit are equal
}

// New creates a new VCSIntegrator instance with the provided configuration.
func New(pluginName, action string, logger hclog.Logger) *VCSIntegrator {
	return &VCSIntegrator{
		PluginName: pluginName,
		Action:     action,
		logger:     logger,
	}
}

// createListRequest creates a VCSListRepositoriesRequest with the specified parameters.
func (i *VCSIntegrator) createListRequest(repo shared.RepositoryParams, language string) shared.VCSListRepositoriesRequest {
	return shared.VCSListRepositoriesRequest{
		VCSRequestBase: shared.VCSRequestBase{
			RepoParam: repo,
			Action:    i.Action,
		},
		Language: language,
	}
}

// createCheckPRRequest creates a VCSRetrievePRInformationRequest with the specified parameters.
func (i *VCSIntegrator) createCheckPRRequest(repo shared.RepositoryParams) shared.VCSRetrievePRInformationRequest {
	return shared.VCSRetrievePRInformationRequest{
		VCSRequestBase: shared.VCSRequestBase{
			RepoParam: shared.RepositoryParams{
				Domain:        repo.Domain,
				Namespace:     repo.Namespace,
				Repository:    repo.Repository,
				PullRequestID: repo.PullRequestID,
			},
			Action: i.Action,
		},
	}
}

// createAddCommentRequest creates a VCSAddCommentToPRRequest with the specified parameters.
func (i *VCSIntegrator) createAddCommentRequest(repo shared.RepositoryParams, options *RunOptionsIntegrationVCS) shared.VCSAddCommentToPRRequest {
	return shared.VCSAddCommentToPRRequest{
		VCSRequestBase: shared.VCSRequestBase{
			RepoParam: shared.RepositoryParams{
				Domain:        repo.Domain,
				Namespace:     repo.Namespace,
				Repository:    repo.Repository,
				PullRequestID: repo.PullRequestID,
			},
			Action: i.Action,
		},
		Comment:   options.Comment,
		FilePaths: options.AttachFiles,
	}
}

// createAddRoleToPRRequest creates a VCSAddRoleToPRRequest with the specified parameters.
func (i *VCSIntegrator) createAddRoleToPRRequest(repo shared.RepositoryParams, options *RunOptionsIntegrationVCS) shared.VCSAddRoleToPRRequest {
	return shared.VCSAddRoleToPRRequest{
		VCSRequestBase: shared.VCSRequestBase{
			RepoParam: shared.RepositoryParams{
				Domain:        repo.Domain,
				Namespace:     repo.Namespace,
				Repository:    repo.Repository,
				PullRequestID: repo.PullRequestID,
			},
			Action: i.Action,
		},
		Login: options.Login,
		Role:  options.Role,
	}
}

// createSetStatusOfPRRequest creates a VCSSetStatusOfPRRequest with the specified parameters.
func (i *VCSIntegrator) createSetStatusOfPRRequest(repo shared.RepositoryParams, options *RunOptionsIntegrationVCS) shared.VCSSetStatusOfPRRequest {
	return shared.VCSSetStatusOfPRRequest{
		VCSRequestBase: shared.VCSRequestBase{
			RepoParam: shared.RepositoryParams{
				Domain:        repo.Domain,
				Namespace:     repo.Namespace,
				Repository:    repo.Repository,
				PullRequestID: repo.PullRequestID,
			},
			Action: i.Action,
		},
		Comment:        options.Comment,
		Login:          options.Login,
		Status:         options.Status,
		LocalTipCommit: options.LocalTipCommit,
	}
}

// PrepIntegrationRequest prepares the integration request based on the specified action.
func (i *VCSIntegrator) PrepIntegrationRequest(cfg *config.Config, options *RunOptionsIntegrationVCS, repo shared.RepositoryParams) (interface{}, error) {
	var arguments interface{}

	switch i.Action {
	case VCSListing:
		arguments = i.createListRequest(repo, options.Language)
	case VCSCheckPR:
		arguments = i.createCheckPRRequest(repo)
	case VCSCommentPR:
		arguments = i.createAddCommentRequest(repo, options)
	case VCSAddRoleToPR:
		arguments = i.createAddRoleToPRRequest(repo, options)
	case VCSSetStatusOfPR:
		arguments = i.createSetStatusOfPRRequest(repo, options)
	default:
		return arguments, fmt.Errorf("action is not implemented: %v", i.Action)
	}

	return arguments, nil
}

// performAction executes the specified action using the VCS plugin.
func performAction(vcsPlugin shared.VCS, options interface{}, action string) (interface{}, error) {
	switch action {
	case VCSListing:
		listRequest, ok := options.(shared.VCSListRepositoriesRequest)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for action '%v'", VCSListing)
		}
		return vcsPlugin.ListRepositories(listRequest)
	case VCSCheckPR:
		checkPRRequest, ok := options.(shared.VCSRetrievePRInformationRequest)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for action '%v'", VCSCheckPR)
		}
		return vcsPlugin.RetrievePRInformation(checkPRRequest)
	case VCSCommentPR:
		addCommentRequest, ok := options.(shared.VCSAddCommentToPRRequest)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for action '%v'", VCSCommentPR)
		}
		return vcsPlugin.AddCommentToPR(addCommentRequest)
	case VCSAddRoleToPR:
		addRoleRequest, ok := options.(shared.VCSAddRoleToPRRequest)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for action '%v'", VCSAddRoleToPR)
		}
		return vcsPlugin.AddRoleToPR(addRoleRequest)
	case VCSSetStatusOfPR:
		setStatusRequest, ok := options.(shared.VCSSetStatusOfPRRequest)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for action '%v'", VCSSetStatusOfPR)
		}
		return vcsPlugin.SetStatusOfPR(setStatusRequest)
	default:
		return nil, fmt.Errorf("unsupported action: %q", action)
	}
}

// IntegrationAction executes the integration action using the VCS plugin.
func (i *VCSIntegrator) IntegrationAction(cfg *config.Config, actionRequest interface{}) (shared.GenericLaunchesResult, error) {
	i.logger.Info("vcs integrator action starting", "action", i.Action)

	var result shared.GenericLaunchesResult
	err := shared.WithPlugin(cfg, i.logger, shared.PluginTypeVCS, i.PluginName, func(raw interface{}) error {
		vcsPlugin, ok := raw.(shared.VCS)
		if !ok {
			return fmt.Errorf("invalid plugin type")
		}

		var err error
		actionResult, err := performAction(vcsPlugin, actionRequest, i.Action)
		if err != nil {
			result.Launches = append(result.Launches, shared.GenericResult{Args: actionRequest, Result: actionResult, Status: "FAILED", Message: err.Error()})
			i.logger.Error("VCS plugin integration failed", "action", i.Action, "actionArgs", actionRequest, "error", err)
			return fmt.Errorf("VCS plugin integration failed. Error: %w", err)
		}
		result.Launches = append(result.Launches, shared.GenericResult{Args: actionRequest, Result: actionResult, Status: "OK", Message: ""})
		return nil
	})

	return result, err
}
