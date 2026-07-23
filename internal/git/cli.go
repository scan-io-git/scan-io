package git

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// execGit runs a git command in repoDir with env appended to the current process
// environment, returning trimmed stdout, trimmed stderr, and the run error. Env
// is never logged (may contain credentials).
func (c *Client) execGit(ctx context.Context, repoDir string, env []string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	c.logger.Debug("running git", "args", args, "dir", repoDir)
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// runGit runs a git command and treats any non-zero exit as a failure: stderr is
// logged at warn and the error is wrapped. Use for commands that must succeed.
func (c *Client) runGit(ctx context.Context, repoDir string, env []string, args ...string) (string, error) {
	out, stderr, err := c.execGit(ctx, repoDir, env, args...)
	if err != nil {
		c.logger.Warn("git command failed", "args", args, "stderr", stderr, "error", err)
		return "", fmt.Errorf("git %s: %w (stderr: %s)", strings.Join(args, " "), err, stderr)
	}
	return out, nil
}

// runGitProbe runs git for a boolean check where a non-zero exit is an expected,
// meaningful outcome rather than a failure — e.g. "git merge-base" exits 1 when
// there is no common ancestor. Non-zero exits are logged at debug, not warn, so
// routine negative results do not surface as alarming log noise.
func (c *Client) runGitProbe(ctx context.Context, repoDir string, env []string, args ...string) (string, error) {
	out, stderr, err := c.execGit(ctx, repoDir, env, args...)
	if err != nil {
		c.logger.Debug("git probe returned non-zero exit", "args", args, "stderr", stderr, "error", err)
		return "", err
	}
	return out, nil
}

// resolveRemoteNameCLI mirrors resolveRemoteName via the git binary: prefer
// "origin", fall back to the first configured remote.
func (c *Client) resolveRemoteNameCLI(ctx context.Context, repoPath string, env []string) (string, error) {
	if out, err := c.runGit(ctx, repoPath, env, "remote", "get-url", origin); err == nil && out != "" {
		return origin, nil
	}
	list, err := c.runGit(ctx, repoPath, env, "remote")
	if err != nil || list == "" {
		return "", fmt.Errorf("no remotes configured in repository")
	}
	return strings.Fields(list)[0], nil
}

// gitCLIEnv returns environment variables for the git CLI that mirror the
// authentication configured on the Client. Returns an error when CLI auth is
// unsupported for this credential type; callers should treat that as a skip.
// The returned env slice is never logged (may contain credentials).
func (c *Client) gitCLIEnv() ([]string, error) {
	env := []string{"GIT_TERMINAL_PROMPT=0"}

	switch c.auth.(type) {
	case *gitssh.PublicKeysCallback:
		// SSH_AUTH_SOCK is inherited from the process env; just suppress prompts.
		env = append(env,
			"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o BatchMode=yes",
		)

	case *gitssh.PublicKeys:
		if c.sshKeyHasPassphrase {
			return nil, fmt.Errorf("ssh-key with passphrase not supported for git CLI; skipping merge-base")
		}
		env = append(env, fmt.Sprintf(
			"GIT_SSH_COMMAND=ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o BatchMode=yes",
			c.sshKeyPath,
		))

	case *githttp.BasicAuth:
		a := c.auth.(*githttp.BasicAuth)
		if a.Username != "" && a.Password != "" {
			creds := base64.StdEncoding.EncodeToString([]byte(a.Username + ":" + a.Password))
			env = append(env,
				"GIT_CONFIG_COUNT=1",
				"GIT_CONFIG_KEY_0=http.extraHeader",
				"GIT_CONFIG_VALUE_0=Authorization: Basic "+creds,
			)
		}
		if InsecureFromCfg(c.globalConfig) {
			env = append(env, "GIT_SSL_NO_VERIFY=true")
		}
	}

	return env, nil
}
