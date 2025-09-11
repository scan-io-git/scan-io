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

	log "github.com/scan-io-git/scan-io/pkg/shared/logger"
)

const (
	origin       = "origin"
	tmpRefPrefix = "refs/tmp/"
)

// CloneRepository clones or fetches a repository into args.TargetFolder, checks out the requested target (branch/tag/PR/commit)
// and returns the target folder path. It auto-repairs (reclone from scratch) shallow/corrupted repos when AutoRepair is true.
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
		case TargetTag:
			c.logger.Info("cloning tag",
				"dst", targetFolder, "cloneURL", safeLogURL(cloneURL), "branch", target.TagRef)
			repo, err = cloneAtRef(ctx, targetFolder, cloneURL, target.TagRef, c.auth,
				depth, insecure, singleBranch, tagsMode, output)
		case TargetPR:
			c.logger.Info("cloning pull request special reference",
				"dst", targetFolder, "cloneURL", safeLogURL(cloneURL), "pr", target.PRRef)
			repo, err = cloneCustomRef(ctx, targetFolder, cloneURL, target.PRRef, c.auth,
				depth, insecure, tagsMode, output)
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
			"target_pr", target.PRRef, "target_tag", target.TagRef,
			"depth", depth, "singleBranch", singleBranch,
			"tagsMode", TagModeToString(tagsMode),
		)
	}

	if existed {
		reclone := func() (*git.Repository, error) {
			tmp, err := os.MkdirTemp("", "reclone-"+info.FullName+"-*")
			if err != nil {
				return nil, fmt.Errorf("mkdtemp: %w", err)
			}

			switch target.Kind {
			case TargetBranch:
				_, err = cloneAtRef(ctx, tmp, cloneURL, target.BranchRef,
					c.auth, depth, insecure, singleBranch, tagsMode, output)
			case TargetTag:
				_, err = cloneAtRef(ctx, tmp, cloneURL, target.TagRef, c.auth,
					depth, insecure, singleBranch, tagsMode, output)
			case TargetPR:
				_, err = cloneCustomRef(ctx, tmp, cloneURL, target.PRRef, c.auth,
					depth, insecure, tagsMode, output)
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
			depth, insecure, tagsMode, output, c.logger, reclone, args.AutoRepair)
		if err != nil {
			return "", fmt.Errorf("error occurred during fetch: %w", err)

		}
	}

	switch target.Kind {
	case TargetBranch:
		c.logger.Debug("checkout branch", "branch", target.BranchRef)
		if err := checkoutRef(repo, target.BranchRef, args.CleanWorkdir, c.logger); err != nil {
			return "", fmt.Errorf("error occurred during checkout: %w", err)
		}
	case TargetPR:
		c.logger.Debug("checkout PR", "pr", target.PRRef)
		if err := checkoutPR(repo, target.PRRef, args.CleanWorkdir, c.logger); err != nil {
			return "", fmt.Errorf("error occurred during checkout: %w", err)
		}
	case TargetCommit, TargetTag:
		c.logger.Debug("checking out hash", "hash", target.CommitHash.String(), "targetFolder", targetFolder)
		if err := checkoutCommit(repo, target.CommitHash, target.TagRef, args.CleanWorkdir, c.logger); err != nil {
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
				"pr", target.PRRef, "tag", target.TagRef, "to_ref", newHead.Name().String(),
				"to_hash", newHead.Hash().String(), "repo", info.Name,
				"depth", depth, "singleBranch", singleBranch,
				"tagsMode", TagModeToString(tagsMode),
			)
		} else {
			c.logger.Info("fetch complete",
				"path", targetFolder,
				"ref", newHead.Name().String(),
				"target", target.Kind.String(), "branch", target.BranchRef,
				"pr", target.PRRef, "tag", target.TagRef, "hash", newHead.Hash().String(),
				"repo", info.Name, "depth", depth, "singleBranch", singleBranch,
				"tagsMode", TagModeToString(tagsMode),
			)
		}
	}

	return targetFolder, nil
}

// cloneAtRef performs a shallow/full clone at a specific branch or tag, creating the repo without checking out files.
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

// cloneCustomRef initializes an empty repo, adds origin, and fetches a provider-specific PR ref into a remote-tracking ref without checkout.
func cloneCustomRef(ctx context.Context, targetFolder, url string, prRef plumbing.ReferenceName,
	auth transport.AuthMethod, depth int, insecureTLS bool, tags git.TagMode, output io.Writer) (*git.Repository, error) {
	repo, err := git.PlainInit(targetFolder, false)
	if err != nil {
		return nil, err
	}

	if _, err := repo.CreateRemote(&gitconfig.RemoteConfig{Name: origin, URLs: []string{url}}); err != nil {
		return nil, err
	}
	// Map provider PR ref to a remote-tracking ref
	local := remoteTrackingForPR(prRef) // refs/remotes/origin/<provider path>
	rs := gitconfig.RefSpec(fmt.Sprintf("+%s:%s", prRef.String(), local.String()))
	// Fetch with explicit refspec
	if err := repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName:      origin,
		Auth:            auth,
		Depth:           depth,
		InsecureSkipTLS: insecureTLS,
		Progress:        output,
		Tags:            tags,
		RefSpecs:        []gitconfig.RefSpec{rs},
	}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, err
	}
	return repo, nil
}

// cloneCommit initializes an empty repo and fetches a specific commit object into a temporary ref for later checkout.
func cloneCommit(ctx context.Context, targetFolder, url string, hash plumbing.Hash,
	auth transport.AuthMethod, depth int, insecureTLS bool, output io.Writer) (*git.Repository, error) {

	repo, err := git.PlainInit(targetFolder, false)
	if err != nil {
		return nil, err
	}

	_, err = repo.CreateRemote(&gitconfig.RemoteConfig{Name: origin, URLs: []string{url}})
	if err != nil {
		return nil, err
	}

	tmpRef := plumbing.ReferenceName(fmt.Sprintf(tmpRefPrefix+"%s", hash.String()))
	refspec := gitconfig.RefSpec(fmt.Sprintf("+%s:%s", hash.String(), tmpRef))
	err = repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName:      origin,
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
	tmpRef := plumbing.ReferenceName(fmt.Sprintf(tmpRefPrefix+"%s", hash.String()))
	_ = repo.Storer.RemoveReference(tmpRef)
}

// fetchTarget fetches the desired refs for a branch/tag/PR/commit into an existing repo, with retry and optional auto-repair via reclone on object corruption/shallow.
func fetchTarget(ctx context.Context, repo *git.Repository, target Target, auth transport.AuthMethod, depth int,
	insecureTLS bool, tags git.TagMode, output io.Writer, logger hclog.Logger, reclone func() (*git.Repository, error), autoRepair bool) (*git.Repository, error) {

	// Special handling for commit-only target: fetch only if we don't have the object.
	if target.Kind == TargetCommit {
		if _, err := repo.CommitObject(target.CommitHash); err == nil {
			// Already have the object locally; nothing to fetch.
			return repo, nil
		}
	}

	rs, err := refspecsFor(target, origin)
	if err != nil {
		return nil, err
	}

	fo := &git.FetchOptions{
		RemoteName:      origin,
		Auth:            auth,
		Depth:           depth,
		InsecureSkipTLS: insecureTLS,
		Tags:            tags,
		Force:           true,
		RefSpecs:        rs,
		Progress:        output,
	}

	try := func() error {
		if err := repo.FetchContext(ctx, fo); err != nil {
			if errors.Is(err, git.NoErrAlreadyUpToDate) {
				logger.Info("repository already up-to-date")
				return nil
			}
			if isRefNotFound(err) {
				switch target.Kind {
				case TargetBranch:
					return fmt.Errorf("remote branch %q not found: %w", target.BranchRef.String(), err)
				case TargetTag:
					return fmt.Errorf("remote tag %q not found: %w", target.TagRef.String(), err)
				case TargetPR:
					return fmt.Errorf("remote PR ref %q not found: %w (provider may prune PR heads after close/merge); try fetch the base branch/commit)", target.PRRef.String(), err)
				case TargetCommit:
					return fmt.Errorf("commit %q not fetchable by SHA on this remote: %w", target.CommitHash.String(), err)
				default:
					return fmt.Errorf("remote ref not found: %w", err)
				}
			}

			// There is no way to update shallow clone, we need to clone from scratch as shallow or full
			// https://github.com/go-git/go-git/issues/1443
			if isObjectMissing(err) {
				logger.Warn("repository appears shallow/corrupt", "error", err)
				if !autoRepair {
					return ErrRecloneConsent
				}

				newRepo, errReclone := reclone()
				if errReclone != nil {
					return fmt.Errorf("reclone after corruption/shallow failed: %w", errReclone)
				}
				*repo = *newRepo
				return nil

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

// checkoutRef creates/updates a local branch from a fetched remote branch and checks it out; optionally resets and cleans the worktree.
func checkoutRef(repo *git.Repository, ref plumbing.ReferenceName, cleanWorkdir bool, log hclog.Logger) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("error accessing worktree: %w", err)
	}

	remoteRef := plumbing.NewRemoteReferenceName(origin, ref.Short())
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

	if cleanWorkdir {
		log.Info("hard reset and clean folder")
		if err := wt.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			return fmt.Errorf("error occurred during reset: %w", err)
		}

		// Remove untracked files and dirs to guarantee clean repo
		_ = wt.Clean(&git.CleanOptions{Dir: true})
	}
	return nil
}

// checkoutCommit checks out an exact commit (or resolves a tag to its commit first); optionally resets and cleans the worktree.
func checkoutCommit(repo *git.Repository, commitHash plumbing.Hash, tagRef plumbing.ReferenceName, cleanWorkdir bool, log hclog.Logger) error {
	if tagRef != "" {
		r, err := repo.Reference(tagRef, true)
		if err != nil {
			return fmt.Errorf("tag %q not present locally; fetch it first: %w", tagRef, err)
		}
		commitHash = r.Hash()

		if to, err := repo.TagObject(commitHash); err == nil && to != nil {
			if c, err := to.Commit(); err == nil {
				commitHash = c.Hash
			}
		}
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("error accessing worktree: %w", err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{Hash: commitHash, Force: true}); err != nil {
		return fmt.Errorf("error occurred during checkout: %w", err)
	}

	if cleanWorkdir {
		log.Info("hard reset and clean folder")
		if err := wt.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			return fmt.Errorf("error occurred during reset: %w", err)
		}

		_ = wt.Clean(&git.CleanOptions{Dir: true})
	}
	return nil
}

// checkoutPR materializes a local branch for a fetched PR ref and checks it out; optionally resets and cleans the worktree.
func checkoutPR(repo *git.Repository, prRef plumbing.ReferenceName, cleanWorkdir bool, log hclog.Logger) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("error accessing worktree: %w", err)
	}

	remoteRef := remoteTrackingForPR(prRef)
	r, err := repo.Reference(remoteRef, true)
	if err != nil {
		return fmt.Errorf("remote PR ref %q not present locally; error: %w", remoteRef, err)
	}

	localRef := localBranchForPR(prRef)
	if err := repo.Storer.SetReference(plumbing.NewHashReference(localRef, r.Hash())); err != nil {
		return fmt.Errorf("set local PR branch: %w", err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{Branch: localRef, Force: true}); err != nil {
		return fmt.Errorf("checkout PR: %w", err)
	}
	if cleanWorkdir {
		log.Info("hard reset and clean folder")
		if err := wt.Reset(&git.ResetOptions{Mode: git.HardReset}); err != nil {
			return fmt.Errorf("reset: %w", err)
		}
		_ = wt.Clean(&git.CleanOptions{Dir: true})
	}
	return nil
}
