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

// runGit runs a git command in repoDir with the given env appended to the current
// process environment. Stdout is returned trimmed; stderr is logged at warn on
// failure. Env is never logged (may contain credentials).
func (c *Client) runGit(ctx context.Context, repoDir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	c.logger.Debug("running git", "args", args, "dir", repoDir)
	if err := cmd.Run(); err != nil {
		c.logger.Warn("git command failed", "args", args, "stderr", stderr.String(), "error", err)
		return "", fmt.Errorf("git %s: %w (stderr: %s)", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
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
