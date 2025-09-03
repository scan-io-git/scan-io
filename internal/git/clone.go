package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gitsight/go-vcsurl"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/hashicorp/go-hclog"

	gitconfig "github.com/go-git/go-git/v5/config"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"

	log "github.com/scan-io-git/scan-io/pkg/shared/logger"
)

// CloneRepository clones or updates a Git repository into the target folder,
// checking out a branch, PR, or specific commit based on the provided configuration.
func (c *Client) CloneRepository(args *shared.VCSFetchRequest) (string, error) {
	targetFolder := args.TargetFolder
	cloneURL := args.CloneURL
	info, err := vcsurl.Parse(cloneURL)
	if err != nil {
		c.logger.Error("failed to parse VCS URL", "VCSURL", cloneURL, "error", err)
		return "", fmt.Errorf("failed to parse VCS URL: %w", err)
	}

	target, err := determineTarget(args.Branch, cloneURL, c.vcs, args, c.auth)
	if err != nil {
		c.logger.Error("failed to determine target", "error", err, "cloneURL", safeLogURL(cloneURL))
		return "", err
	}

	gitLog := log.GetLoggerOutput(c.logger)
	output := io.MultiWriter(
		gitLog,
		os.Stderr,
	)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	depth := args.Depth
	insecure := InsecureFromCfg(c.globalConfig)
	singleBranch := args.SingleBranch
	tagsMode := args.TagMode
	ciMode := config.IsCI(c.globalConfig)

	c.logger.Debug("start fetch",
		"repo", info.Name, "target", target.Kind.String(), "branch", target.BranchRef,
		"commit", target.CommitHash.String(), "cloneURL", safeLogURL(cloneURL), "dst", targetFolder,
		"pr", target.PRRef, "depth", depth, "singleBranch", singleBranch, "tagsMode", TagModeToString(tagsMode))

	var haveRef string
	var haveHash plumbing.Hash
	repo, openErr := git.PlainOpen(targetFolder)
	existed := (openErr == nil)
	if !existed {
		c.logger.Info("repository not found locally — cloning",
			"dst", targetFolder, "cloneURL", safeLogURL(cloneURL), "target", target.Kind.String(),
			"depth", depth, "singleBranch", singleBranch, "tagsMode", TagModeToString(tagsMode))

		switch target.Kind {
		case TargetBranch:
			c.logger.Info("cloning branch",
				"dst", targetFolder, "cloneURL", safeLogURL(cloneURL), "branch", target.BranchRef)
			repo, err = cloneAtRef(ctx, targetFolder, cloneURL, target.BranchRef, c.auth,
				depth, insecure, singleBranch, tagsMode, output)
		case TargetPR:
			c.logger.Info("cloning pull request",
				"dst", targetFolder, "cloneURL", safeLogURL(cloneURL), "pr", target.PRRef)
			repo, err = cloneAtRef(ctx, targetFolder, cloneURL, target.PRRef, c.auth,
				depth, insecure, singleBranch, tagsMode, output)
		case TargetCommit:
			c.logger.Info("cloning commit",
				"dst", targetFolder, "cloneURL", safeLogURL(cloneURL), "commit", target.CommitHash.String())
			repo, err = cloneCommit(ctx, targetFolder, cloneURL, target.CommitHash,
				c.auth, depth, insecure, output)
		default:
			err = ErrUnsupportedTargetKind
		}
		if err != nil {
			return "", fmt.Errorf("error occurred during clone: %w", err)
		}
	} else {
		if head, herr := repo.Head(); herr == nil {
			haveRef, haveHash = head.Name().String(), head.Hash()
		}

		haveURL := originURL(repo)
		if !sameRemote(haveURL, cloneURL) {
			return "", fmt.Errorf("%w: have %q want %q", ErrDifferentRepo, safeLogURL(haveURL), safeLogURL(cloneURL))
		}

		c.logger.Info("repository exists — updating...",
			"dst", targetFolder,
			"remote", safeLogURL(haveURL),
			"current_ref", haveRef,
			"current_hash", haveHash.String(),
			"target_kind", target.Kind.String(),
			"target_branch", target.BranchRef.String(),
			"pr", target.PRRef, "depth", depth,
			"singleBranch", singleBranch, "tagsMode", TagModeToString(tagsMode),
		)
	}

	if existed {
		reclone := func() (*git.Repository, error) {
			if !ciMode {
				// TODO: get user consent
				return nil, ErrRecloneConsent

			}
			parent := filepath.Dir(targetFolder)
			tmp, err := os.MkdirTemp(parent, ".reclone-*")
			if err != nil {
				return nil, fmt.Errorf("mkdtemp: %w", err)
			}

			switch target.Kind {
			case TargetBranch:
				_, err = cloneAtRef(ctx, tmp, cloneURL, target.BranchRef,
					c.auth, depth, insecure, singleBranch, tagsMode, output)
			case TargetPR:
				_, err = cloneAtRef(ctx, tmp, cloneURL, target.PRRef,
					c.auth, depth, insecure, singleBranch, tagsMode, output)
			case TargetCommit:
				_, err = cloneCommit(ctx, tmp, cloneURL, target.CommitHash,
					c.auth, depth, insecure, output)
			default:
				err = ErrUnsupportedTargetKind
			}
			if err != nil {
				_ = os.RemoveAll(tmp)
				return nil, fmt.Errorf("reclone: %w", err)
			}

			abs, _ := filepath.Abs(targetFolder)
			parentAbs, _ := filepath.Abs(filepath.Dir(targetFolder))
			if !strings.HasPrefix(abs, parentAbs+string(os.PathSeparator)) {
				return nil, fmt.Errorf("refusing to delete unmanaged path: %q", abs)
			}
			if err := os.RemoveAll(targetFolder); err != nil {
				_ = os.RemoveAll(tmp)
				return nil, fmt.Errorf("remove old repo: %w", err)
			}

			if err := os.Rename(tmp, targetFolder); err != nil {
				_ = os.RemoveAll(tmp)
				return nil, fmt.Errorf("activate new repo: %w", err)
			}
			reopened, openErr := git.PlainOpen(targetFolder)
			if openErr != nil {
				return nil, fmt.Errorf("open swapped repo: %w", openErr)
			}
			return reopened, nil
		}

		c.logger.Debug("fetch data", "targetFolder", targetFolder)
		repo, err = fetchTarget(ctx, repo, target, c.auth,
			depth, insecure, tagsMode, output, c.logger, reclone, ciMode)
		if err != nil {
			return "", fmt.Errorf("error occurred during fetch: %w", err)

		}
	}

	switch target.Kind {
	case TargetBranch:
		c.logger.Debug("checkout branch", "branch", target.BranchRef)
		if err := checkoutRef(repo, target.BranchRef, ciMode); err != nil {
			return "", fmt.Errorf("error occurred during checkout: %w", err)
		}
	case TargetPR:
		c.logger.Debug("checkout PR", "pr", target.PRRef)
		if err := checkoutPR(repo, target.PRRef, ciMode); err != nil {
			return "", fmt.Errorf("error occurred during checkout: %w", err)
		}
	case TargetCommit:
		c.logger.Debug("checking out hash", "hash", target.CommitHash.String(), "targetFolder", targetFolder)
		if err := checkoutCommit(repo, target.CommitHash, ciMode); err != nil {
			return "", err
		}
		cleanupTmpRef(repo, target.CommitHash)
	}

	if newHead, err := repo.Head(); err == nil {
		if existed {
			c.logger.Info("update complete",
				"path", targetFolder,
				"from_ref", haveRef, "from_hash", haveHash.String(),
				"target", target.Kind.String(), "branch", target.BranchRef,
				"pr", target.PRRef, "to_ref", newHead.Name().String(),
				"to_hash", newHead.Hash().String(), "repo", info.Name,
				"depth", depth, "singleBranch", singleBranch,
				"tagsMode", TagModeToString(tagsMode),
			)
		} else {
			c.logger.Info("fetch complete",
				"path", targetFolder,
				"ref", newHead.Name().String(),
				"target", target.Kind.String(), "branch", target.BranchRef,
				"pr", target.PRRef, "hash", newHead.Hash().String(),
				"repo", info.Name, "depth", depth, "singleBranch", singleBranch,
				"tagsMode", TagModeToString(tagsMode),
			)
		}
	}

	return targetFolder, nil
}

// cloneAtRef clones a repository at the specified branch or PR reference, without performing a checkout.
func cloneAtRef(ctx context.Context, targetFolder, url string, ref plumbing.ReferenceName,
	auth transport.AuthMethod, depth int, insecureTLS, singleBranch bool, tags git.TagMode, output io.Writer) (*git.Repository, error) {

	return git.PlainCloneContext(ctx, targetFolder, false, &git.CloneOptions{
		Auth:            auth,
		URL:             url,
		ReferenceName:   ref,
		SingleBranch:    singleBranch,
		Tags:            tags,
		Depth:           depth,
		InsecureSkipTLS: insecureTLS,
		Progress:        output,
		NoCheckout:      true,
	})
}

// cloneCommit initializes an empty repository and fetches a specific commit hash for later checkout.
func cloneCommit(ctx context.Context, targetFolder, url string, hash plumbing.Hash,
	auth transport.AuthMethod, depth int, insecureTLS bool, output io.Writer) (*git.Repository, error) {

	repo, err := git.PlainInit(targetFolder, false)
	if err != nil {
		return nil, err
	}

	_, err = repo.CreateRemote(&gitconfig.RemoteConfig{Name: "origin", URLs: []string{url}})
	if err != nil {
		return nil, err
	}

	tmpRef := plumbing.ReferenceName(fmt.Sprintf("refs/tmp/%s", hash.String()))
	refspec := gitconfig.RefSpec(fmt.Sprintf("+%s:%s", hash.String(), tmpRef))
	err = repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName:      "origin",
		Auth:            auth,
		Depth:           depth,
		InsecureSkipTLS: insecureTLS,
		Progress:        output,
		RefSpecs:        []gitconfig.RefSpec{refspec},
	})
	if err != nil {
		return nil, fmt.Errorf("fetch by SHA not supported by remote; need a containing ref: %w", err)
	}
	return repo, nil
}

// cleanupTmpRef removes temporary references used during commit-based cloning to keep the repository clean.
func cleanupTmpRef(repo *git.Repository, hash plumbing.Hash) {
	tmpRef := plumbing.ReferenceName(fmt.Sprintf("refs/tmp/%s", hash.String()))
	_ = repo.Storer.RemoveReference(tmpRef)
}

// fetchTarget fetches updates for the specified branch, PR, or commit, with retry and shallow clone handling.
func fetchTarget(ctx context.Context, repo *git.Repository, target Target, auth transport.AuthMethod, depth int,
	insecureTLS bool, tags git.TagMode, output io.Writer, logger hclog.Logger, reclone func() (*git.Repository, error), ciMode bool) (*git.Repository, error) {

	fo := &git.FetchOptions{
		RemoteName:      "origin",
		Auth:            auth,
		Depth:           depth,
		InsecureSkipTLS: insecureTLS,
		Progress:        output,
		Tags:            tags,
	}

	switch target.Kind {
	case TargetBranch:
		// +refs/heads/X:refs/remotes/origin/X
		rs := gitconfig.RefSpec(fmt.Sprintf("+%s:%s",
			target.BranchRef.String(),
			plumbing.NewRemoteReferenceName("origin", target.BranchRef.Short()).String()))
		fo.RefSpecs = []gitconfig.RefSpec{rs}
	case TargetPR:
		// +<provider PR ref> : refs/remotes/origin/<provider path>
		local := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s",
			strings.TrimPrefix(target.PRRef.String(), "refs/")))
		rs := gitconfig.RefSpec(fmt.Sprintf("+%s:%s", target.PRRef.String(), local.String()))
		fo.RefSpecs = []gitconfig.RefSpec{rs}
	case TargetCommit:
		// For updates of a commit-only repo we can no-op
		return repo, nil
	default:
		return nil, fmt.Errorf("unsupported target kind")
	}

	try := func() error {
		if err := repo.FetchContext(ctx, fo); err != nil {
			if errors.Is(err, git.NoErrAlreadyUpToDate) {
				logger.Info("repository already up-to-date")
				return nil
			}
			if isRefNotFound(err) {
				if target.Kind == TargetPR {
					return fmt.Errorf("remote PR ref %q not found: %w (note: some providers prune PR/MR head refs after close/merge; try fetch the base branch/commit)",
						target.PRRef.String(), err)
				}
				return fmt.Errorf("remote ref not found for %q: %w", target.BranchRef.String(), err)
			}

			if isObjectMissing(err) {
				if ciMode {
					hard := *fo
					hard.Force = true
					hard.Prune = true
					hard.Depth = 0
					if errFetch := repo.FetchContext(ctx, &hard); errFetch == nil || errors.Is(errFetch, git.NoErrAlreadyUpToDate) {
						return nil
					}
					logger.Warn("repository appears shallow/corrupt; recloning", "error", err)
					newRepo, errReclone := reclone()
					if errReclone != nil {
						return fmt.Errorf("reclone after corruption failed: %w (original fetch error: %v)", errReclone, err)
					}
					*repo = *newRepo
					return nil
				}
				return ErrRecloneConsent
			}
			return err
		}
		return nil
	}

	if err := try(); err != nil {
		d := time.Duration(100+rand.Intn(150)) * time.Millisecond
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		logger.Warn("fetch failed; retrying once", "error", err)
		if err2 := try(); err2 != nil {
			return nil, fmt.Errorf("fetch failed after retry: %w", err2)
		}
	}

	return repo, nil
}

// checkoutRef checks out a branch reference and optionally resets/cleans the repository in CI mode.
func checkoutRef(repo *git.Repository, ref plumbing.ReferenceName, ciMode bool) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("error accessing worktree: %w", err)
	}

	remoteRef := plumbing.NewRemoteReferenceName("origin", ref.Short())
	r, err := repo.Reference(remoteRef, true)
	if err != nil {
		return fmt.Errorf("remote branch %q not present locally; fetch it first: %w", remoteRef, err)
	}

	localRef := plumbing.NewBranchReferenceName(ref.Short())
	if err := repo.Storer.SetReference(plumbing.NewHashReference(localRef, r.Hash())); err != nil {
		return fmt.Errorf("failed to set local branch ref: %w", err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{Branch: localRef, Force: true}); err != nil {
		return fmt.Errorf("error occurred during checkout: %w", err)
	}

	if ciMode {
		if err := wt.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			return fmt.Errorf("error occurred during reset: %w", err)
		}

		// Remove untracked files and dirs to guarantee clean repo
		_ = wt.Clean(&git.CleanOptions{Dir: true})
	}
	return nil
}

// checkoutCommit checks out a specific commit hash and optionally resets/cleans the repository in CI mode.
func checkoutCommit(repo *git.Repository, commitHash plumbing.Hash, ciMode bool) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("error accessing worktree: %w", err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{Hash: commitHash, Force: true}); err != nil {
		return fmt.Errorf("error occurred during checkout: %w", err)
	}

	if ciMode {
		if err := wt.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			return fmt.Errorf("error occurred during reset: %w", err)
		}

		_ = wt.Clean(&git.CleanOptions{Dir: true})
	}
	return nil
}

// checkoutPR checks out a pull request reference as a local branch and resets/cleans in CI mode if configured.
func checkoutPR(repo *git.Repository, prRef plumbing.ReferenceName, ciMode bool) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("error accessing worktree: %w", err)
	}

	remoteRef := remoteTrackingForPR(prRef)
	r, err := repo.Reference(remoteRef, true)
	if err != nil {
		return fmt.Errorf("remote PR ref %q not present locally; fetch it first: %w", remoteRef, err)
	}

	localRef := localBranchForPR(prRef)
	if err := repo.Storer.SetReference(plumbing.NewHashReference(localRef, r.Hash())); err != nil {
		return fmt.Errorf("set local PR branch: %w", err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{Branch: localRef, Force: ciMode}); err != nil {
		return fmt.Errorf("checkout PR: %w", err)
	}
	if ciMode {
		if err := wt.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			return fmt.Errorf("reset: %w", err)
		}
		_ = wt.Clean(&git.CleanOptions{Dir: true})
	}
	return nil
}
