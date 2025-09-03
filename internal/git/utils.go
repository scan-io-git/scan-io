package git

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/scan-io-git/scan-io/pkg/shared"

	ftutils "github.com/scan-io-git/scan-io/internal/fetcher-utils"
	scconfig "github.com/scan-io-git/scan-io/pkg/shared/config"
)

type TargetKind int

const (
	TargetDefault TargetKind = iota
	TargetBranch
	TargetPR
	TargetCommit
)

type Target struct {
	Kind       TargetKind
	BranchRef  plumbing.ReferenceName // refs/heads/...
	PRRef      plumbing.ReferenceName // provider PR ref (e.g. refs/pull/123/head)
	CommitHash plumbing.Hash          // SHA hash
}

func (t TargetKind) String() string {
	switch t {
	case TargetBranch:
		return "branch"
	case TargetPR:
		return "pull-request"
	case TargetCommit:
		return "commit"
	case TargetDefault:
		return "default"
	default:
		return "unknown"
	}
}

// determineTarget determines the type of target (branch, PR, commit, or default branch) for the clone operation.
func determineTarget(branchOrCommit, cloneURL, vcs string, args *shared.VCSFetchRequest, auth transport.AuthMethod) (Target, error) {
	var t Target

	// PR fetch case
	if args.FetchMode == ftutils.PRRefMode && args.RepoParam.PullRequestID != "" {
		t.Kind = TargetPR

		head, _, ok := prRefsForVCS(vcs, args.RepoParam.PullRequestID)
		if !ok {
			return t, fmt.Errorf("vcs %q not supported for pr scanning", vcs)
		}

		t.PRRef = head
		return t, nil
	}

	// If the branch is explicitly provided, return it as the reference
	if s := strings.TrimSpace(branchOrCommit); s != "" {
		// Commit hash case
		if plumbing.IsHash(s) {
			if len(s) != 40 {
				return t, ErrShortCommitSHA
			}
			t.Kind = TargetCommit
			t.CommitHash = plumbing.NewHash(s)
			return t, nil
		}
		// Branch name case
		rn := plumbing.ReferenceName(s)

		// Ensure we avoid double concatenation of refs if it already looks like a ref
		if !rn.IsBranch() && !rn.IsRemote() && !rn.IsTag() && !rn.IsNote() {
			rn = plumbing.NewBranchReferenceName(rn.String())
		}
		t.Kind = TargetBranch
		t.BranchRef = rn
		return t, nil
	}

	// No branch provided, resolve the default branch by fetching the remote HEAD
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{cloneURL},
	})

	refs, err := remote.List(&git.ListOptions{Auth: auth})
	if err != nil {
		return t, fmt.Errorf("list remote refs: %w", err)
	}

	// Find the reference that HEAD points to (default branch)
	for _, ref := range refs {
		if ref.Name() == plumbing.HEAD {
			target := ref.Target()
			if target.IsBranch() {
				t.Kind = TargetBranch
				t.BranchRef = target // default branch (refs/heads/main or refs/heads/master)
				return t, nil
			}
		}
	}
	return t, ErrDefaultBranchHead
}

// findGitRepositoryPath walks up the directory tree to find the root of a Git repository.
func findGitRepositoryPath(sourceFolder string) (string, error) {
	if sourceFolder == "" {
		return "", fmt.Errorf("source folder is not set")
	}

	// check if source folder is a subfolder of a git repository
	for {
		_, err := git.PlainOpen(sourceFolder)
		if err == nil {
			return sourceFolder, nil
		}

		// move up one level
		sourceFolder = filepath.Dir(sourceFolder)

		// check if reached the root folder
		if sourceFolder == filepath.Dir(sourceFolder) {
			break
		}
	}

	return "", fmt.Errorf("source folder is not a git repository")
}

// prRefsForVCS returns the provider-specific reference names (head and merge) for a pull request or merge request.
func prRefsForVCS(vcs, id string) (head, merge plumbing.ReferenceName, ok bool) {
	switch vcs {
	case "github":
		// refs/pull/<ID>/head (the PR tip) and refs/pull/<ID>/merge (synthetic merge)
		return plumbing.ReferenceName(fmt.Sprintf("refs/pull/%s/head", id)),
			plumbing.ReferenceName(fmt.Sprintf("refs/pull/%s/merge", id)), true
	case "gitlab":
		// refs/merge-requests/<ID>/head and refs/merge-requests/<ID>/merge
		return plumbing.ReferenceName(fmt.Sprintf("refs/merge-requests/%s/head", id)),
			plumbing.ReferenceName(fmt.Sprintf("refs/merge-requests/%s/merge", id)), true
	case "bitbucket":
		// refs/pull-requests/<ID>/from (source tip) and refs/pull-requests/<ID>/merge (synthetic merge)
		return plumbing.ReferenceName(fmt.Sprintf("refs/pull-requests/%s/from", id)),
			plumbing.ReferenceName(fmt.Sprintf("refs/pull-requests/%s/merge", id)), true
	default:
		return "", "", false
	}
}

// originURL returns the remote URL of the origin remote for the repository.
func originURL(repo *git.Repository) string {
	r, err := repo.Remote("origin")
	if err != nil || r == nil || len(r.Config().URLs) == 0 {
		return ""
	}
	return r.Config().URLs[0]
}

// sameRemote checks whether two remote URLs refer to the same repository, ignoring schemes, casing, and .git suffix.
func sameRemote(a, b string) bool {
	ha, pa, ea := normalizeRemote(a)
	hb, pb, eb := normalizeRemote(b)
	if ea != nil || eb != nil {
		// Fall back to conservative string compare without creds/query/fragment.
		strip := func(u string) string {
			u = strings.TrimSuffix(u, ".git")
			u = strings.TrimRight(u, "/")
			if p, err := url.Parse(u); err == nil && p.Scheme != "" {
				p.User = nil
				p.RawQuery, p.Fragment = "", ""
				u = p.String()
			}
			return strings.ToLower(u)
		}
		return strip(a) == strip(b)
	}
	return ha == hb && pa == pb
}

func normalizeRemote(raw string) (string, string, error) {
	u, err := vcsurl.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("parse remote: %w", err)
	}

	host := strings.ToLower(strings.TrimSpace(string(u.Host)))
	full := u.FullName
	if full == "" && u.Username != "" && u.Name != "" {
		full = u.Username + "/" + u.Name
	}
	full = strings.ToLower(strings.TrimSuffix(full, ".git"))
	return host, full, nil
}

// remoteTrackingForPR constructs the remote-tracking reference path for a pull request.
func remoteTrackingForPR(remotePRRef plumbing.ReferenceName) plumbing.ReferenceName {
	// refs/remotes/origin/<provider path>
	suffix := strings.TrimPrefix(remotePRRef.String(), "refs/")
	return plumbing.ReferenceName("refs/remotes/origin/" + suffix)
}

// localBranchForPR constructs the local branch reference path for a pull request.
func localBranchForPR(remotePRRef plumbing.ReferenceName) plumbing.ReferenceName {
	// refs/heads/<provider path>
	suffix := strings.TrimPrefix(remotePRRef.String(), "refs/")
	return plumbing.ReferenceName("refs/heads/" + suffix)
}

// isObjectMissing detects object-missing errors in go-git operations, independent of error string variations.
func isObjectMissing(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "object not found") || strings.Contains(s, "missing blob") || strings.Contains(s, "missing tree")
}

// isRefNotFound detects missing reference errors from go-git fetch or reference lookups.
func isRefNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "reference not found") || strings.Contains(s, "couldn't find remote ref")
}

// safeLogURL redacts sensitive credentials (e.g., tokens or passwords) from Git URLs for safe logging.
func safeLogURL(u string) string {
	if parsed, err := url.Parse(u); err == nil {
		return parsed.Redacted()
	}
	return u
}

// insecureFromCfg returns true if TLS verification should be skipped, based on configuration.
func InsecureFromCfg(cfg *scconfig.Config) bool {
	return scconfig.GetBoolValue(cfg.GitClient, "InsecureTLS", false)
}

// TagModeToString converts a git.TagMode value to a human-readable string.
func TagModeToString(mode git.TagMode) string {
	switch mode {
	case git.AllTags:
		return "all"
	case git.TagFollowing:
		return "follow"
	case git.NoTags:
		return "no"
	default:
		return fmt.Sprintf("unknown(%d)", mode)
	}
}
