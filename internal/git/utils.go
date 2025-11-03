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

	ftutils "github.com/scan-io-git/scan-io/internal/fetcherutils"
	scconfig "github.com/scan-io-git/scan-io/pkg/shared/config"
)

// TargetKind represents the type of target to fetch from a repository.
type TargetKind int

const (
	// TargetDefault indicates no specific target provided — defaults to repository's default branch.
	TargetDefault TargetKind = iota
	// TargetBranch indicates a branch reference (e.g., refs/heads/main).
	TargetBranch
	// TargetNote indicates a note reference (e.g., refs/notes). Not implemented
	TargetNote
	// TargetTag indicates a tag reference (e.g., refs/tags).
	TargetTag
	// TargetPR indicates a pull request reference (e.g., refs/pull/123/head).
	TargetPR
	// TargetCommit indicates a direct commit hash target.
	TargetCommit
)

// Target represents a repository target for fetch/clone operations.
// It specifies what to fetch: branch, pull request, or commit.
type Target struct {
	Kind       TargetKind
	BranchRef  plumbing.ReferenceName // refs/heads/...
	TagRef     plumbing.ReferenceName // refs/tags/...
	NotesRef   plumbing.ReferenceName // refs/notes/...
	PRRef      plumbing.ReferenceName // provider PR ref (e.g. refs/pull/123/head)
	CommitHash plumbing.Hash          // SHA hash
}

// String returns the human-readable string representation of a TargetKind.
func (t TargetKind) String() string {
	switch t {
	case TargetBranch:
		return "branch"
	case TargetTag:
		return "tag"
	case TargetNote:
		return "note"
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

// determineTarget resolves the desired fetch/checkout target from user input (branch/commit/PR/tag), remote refs, and VCS provider semantics.
func determineTarget(branchOrCommit, cloneURL, vcs string, args *shared.VCSFetchRequest, auth transport.AuthMethod) (Target, error) {
	var t Target

	remoteRefs, err := listRemoteRefs(cloneURL, auth)
	if err != nil {
		return t, fmt.Errorf("list remote refs: %w", err)
	}
	idxRemoteRef := indexRefs(remoteRefs)

	// PR fetch case
	if args.FetchMode == ftutils.PRRefMode && args.RepoParam.PullRequestID != "" {
		head, _, ok := prRefsForVCS(vcs, args.RepoParam.PullRequestID)
		if !ok {
			return t, fmt.Errorf("vcs %q not supported for pr scanning", vcs)
		}

		if _, ok := idxRemoteRef[head]; !ok {
			return t, fmt.Errorf("remote ref %q not found", head.String())
		}

		return Target{Kind: TargetPR, PRRef: head}, nil
	}

	// If the branch is explicitly provided, return it as the reference
	if s := strings.TrimSpace(branchOrCommit); s != "" {
		// Commit hash case
		if plumbing.IsHash(s) {
			if len(s) != 40 {
				return t, ErrShortCommitSHA
			}

			return Target{Kind: TargetCommit, CommitHash: plumbing.NewHash(s)}, nil
		}
		// Branch name case
		rn := plumbing.ReferenceName(s)

		if strings.HasPrefix(s, "refs/") {
			switch {
			case rn.IsBranch(): // refs/heads/..
				if _, ok := idxRemoteRef[rn]; !ok {
					return t, fmt.Errorf("remote branch %q not found", rn)
				}
				return Target{Kind: TargetBranch, BranchRef: rn}, nil
			case rn.IsTag(): // refs/tags/...
				if _, ok := idxRemoteRef[rn]; !ok {
					return t, fmt.Errorf("remote tag %q not found", rn)
				}
				return Target{Kind: TargetTag, TagRef: rn}, nil
			// case rn.IsRemote(): // refs/remotes/...
			// if _, ok := idxRemoteRef[rn]; !ok {
			// 	return t, fmt.Errorf("remote branch %q not found", rn)
			// }
			// 	return Target{Kind: TargetPR, PRRef: rn}, nil
			case rn.IsNote(): // refs/notes/…
				return Target{Kind: TargetDefault}, fmt.Errorf("note not supported as fetch target")
			default:
				// Treat any other refs/* (e.g., PR/MR namespaces) as custom PR refs
				if _, ok := idxRemoteRef[rn]; !ok {
					return t, fmt.Errorf("remote reference %q not found", rn)
				}
				return Target{Kind: TargetPR, PRRef: rn}, nil
			}
		}

		// Ensure we avoid double concatenation of refs if it already looks like a ref
		if tt, ok := pickRefByPrecedenceIdx(idxRemoteRef, s); ok {
			return tt, nil
		}

		return t, fmt.Errorf("no branch or tag named %q found on remote", s)
	}

	// Find the reference that HEAD points to (default branch)
	for _, ref := range remoteRefs {
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

// listRemoteRefs returns all advertised references for the remote at cloneURL using the provided auth.
func listRemoteRefs(cloneURL string, auth transport.AuthMethod) ([]*plumbing.Reference, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: origin,
		URLs: []string{cloneURL},
	})
	return remote.List(&git.ListOptions{Auth: auth})
}

// indexRefs builds a set of reference names from a remote ref list for 0(1) checks.
func indexRefs(refs []*plumbing.Reference) map[plumbing.ReferenceName]struct{} {
	m := make(map[plumbing.ReferenceName]struct{}, len(refs))
	for _, r := range refs {
		m[r.Name()] = struct{}{}
	}
	return m
}

// pickRefByPrecedenceIdx prefers a branch over a tag when both exist for the same short name; returns Target and true on match.
func pickRefByPrecedenceIdx(idx map[plumbing.ReferenceName]struct{}, short string) (Target, bool) {
	br := plumbing.NewBranchReferenceName(short)
	if _, ok := idx[br]; ok {
		return Target{Kind: TargetBranch, BranchRef: br}, true
	}
	tg := plumbing.NewTagReferenceName(short)
	if _, ok := idx[tg]; ok {
		return Target{Kind: TargetTag, TagRef: tg}, true
	}
	return Target{}, false
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

// refspecsFor builds the minimal fetch refspecs for the given target and remote name (branch/tag/PR/commit).
func refspecsFor(target Target, remoteName string) ([]config.RefSpec, error) {
	switch target.Kind {
	case TargetBranch:
		// +refs/heads/X:refs/remotes/origin/X
		src := target.BranchRef.String()
		dst := plumbing.NewRemoteReferenceName(remoteName, target.BranchRef.Short()).String()
		return []config.RefSpec{config.RefSpec("+" + src + ":" + dst)}, nil

	case TargetPR:
		// +refs/pull/1/head : refs/remotes/origin/pull/1/head
		src := target.PRRef.String()
		dst := "refs/remotes/" + remoteName + "/" + strings.TrimPrefix(src, "refs/")
		return []config.RefSpec{config.RefSpec("+" + src + ":" + dst)}, nil

	case TargetTag:
		// +refs/tags/X : refs/tags/X
		src := target.TagRef.String()
		dst := src
		return []config.RefSpec{config.RefSpec("+" + src + ":" + dst)}, nil

	case TargetCommit:
		// Only if we need to ensure the object exists. If already present locally, no refspecs are needed.
		tmp := plumbing.ReferenceName(tmpRefPrefix + target.CommitHash.String())
		src := target.CommitHash.String()
		dst := tmp.String()
		return []config.RefSpec{config.RefSpec("+" + src + ":" + dst)}, nil

	default:
		return nil, fmt.Errorf("unsupported target kind: %v", target.Kind.String())
	}
}

// originURL returns the remote URL of the origin remote for the repository.
func originURL(repo *git.Repository) string {
	r, err := repo.Remote(origin)
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

// normalizeRemote parses and normalizes a Git remote URL to extract the host and full repository path.
// It removes protocol, credentials, query parameters, and the `.git` suffix to ensure consistent comparison.
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

// remoteTrackingForPR maps a provider PR ref (e.g., refs/pull/1/head) to its corresponding remote-tracking ref under refs/remotes/origin/.
func remoteTrackingForPR(remotePRRef plumbing.ReferenceName) plumbing.ReferenceName {
	// refs/remotes/origin/<provider path>
	suffix := strings.TrimPrefix(remotePRRef.String(), "refs/")
	return plumbing.ReferenceName("refs/remotes/" + origin + "/" + suffix)
}

// localBranchForPR maps a provider PR ref to a local branch name under refs/heads/ for checkout.
func localBranchForPR(remotePRRef plumbing.ReferenceName) plumbing.ReferenceName {
	// refs/heads/<provider path>
	suffix := strings.TrimPrefix(remotePRRef.String(), "refs/")
	return plumbing.ReferenceName("refs/heads/" + suffix)
}

// isObjectMissing heuristically detects go-git errors indicating missing/corrupted objects or shallow history issues.
func isObjectMissing(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "object not found") ||
		strings.Contains(s, "missing blob") ||
		strings.Contains(s, "missing tree") ||
		strings.Contains(s, "delta base") ||
		strings.Contains(s, "no such object") ||
		strings.Contains(s, "invalid commit") ||
		(strings.Contains(s, "want ") && strings.Contains(s, "not valid"))
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
	return scconfig.SetThenPtr(cfg.GitClient.InsecureTLS, false)
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

func NormalizeFullHash(raw string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if len(trimmed) != 40 {
		return "", fmt.Errorf("commit hash must be a 40-character SHA-1, got %d characters", len(trimmed))
	}
	for _, r := range trimmed {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return "", fmt.Errorf("commit hash must be hexadecimal, got %q", raw)
		}
	}
	return trimmed, nil
}
