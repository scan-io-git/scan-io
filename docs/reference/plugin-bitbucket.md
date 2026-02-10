# Bitbucket Plugin

The Bitbucket plugin provides comprehensive support for interacting with Bitbucket version control systems. It offers a range of functionalities designed to streamline repository management and enhance CI/CD workflows, with a strong focus on security-related processes.

## Table of Contents
- [Supported Versions of API](#supported-versions-of-api)
- [Supported Actions](#supported-actions)
- [Supported Authentication Types](#supported-authentication-types)
- [Configuration References](#configuration-references)
   - [Configuration File](#configuration-file)
   - [Environment Variables](#environment-variables)
- [Usage Examples](#usage-examples)
   - [Command List](#command-list)
     - [Setup Prerequisites](#setup-prerequisites)
     - [Validation](#validation)
     - [Supported URL Types](#supported-url-types)
     - [Actions](#actions)
   - [Command Fetch](#command-fetch)
     - [Setup Prerequisites](#setup-prerequisites-1)
     - [Validation](#validation-1)
     - [Supported URL Types](#supported-url-types-1)
     - [Actions](#actions-1)
- [Known Issues and Fixes](#known-issues-and-fixes)


## Supported Versions of API
| API Version               | Supported | Note                                                                            |
|---------------------------|-----------|---------------------------------------------------------------------------------|
| APIv1                     |     ✅    |  This version is still used for on-premises Bitbucket installations. However, it was permanently removed from the Bitbucket Cloud REST API on April 29, 2019. [Learn more](https://support.atlassian.com/bitbucket-cloud/docs/use-bitbucket-rest-api-version-1/).|
| Cloud REST API            |     ❌    | This is the current version used in Bitbucket Cloud installations. [Learn more](https://developer.atlassian.com/server/bitbucket/rest/v905/intro/#about).|

## Supported Actions
| Action                                        | Command | Supported  |
|-----------------------------------------------|---------|------------|
| List all available repositories in a VCS      |   list  |     ✅     |
| List repositories within a namespace          |   list  |     ✅     |
| Filter repositories by programming language   |   list  |     ❌     |
| List repositories in a user namespace         |   list  |     ✅     |
| Fetch a specific repository                   |  fetch  |     ✅     |
| Fetch a specific pull request                 |  fetch  |     ✅     |
| Fetch repositories in bulk from an input file |  fetch  |     ✅     |

## Supported Authentication Types
| Authentication Type   | Command       | Supported  |
|-----------------------|---------------|------------|
| SSH key               |     fetch     |     ✅     |
| SSH agent             |     fetch     |     ✅     |
| HTTP                  |  list, fetch  |     ✅     |


## Bitbucket Plugin
### Configuration File
The Bitbucket plugin enables Scanio to interact with Bitbucket repositories for operations such as listing repositories, cloning code, and managing pull requests. This plugin uses the `bitbucket_plugin` directive, which supports the following settings.

| Directive          | Default Value | Description                                                                                              |
|---------------------|---------------|----------------------------------------------------------------------------------------------------------|
| `username`         | `none`          | Bitbucket username for authentication. **Mandatory** for all Bitbucket operations.                      |
| `token`            | `none`          | Bitbucket access token for authentication.                                                              |
| `ssh_key_password` | `none`          | Password for the SSH key used in Bitbucket operations (e.g., fetch command with `auth-type=ssh-key`).   |

### Environment Variables
The Bitbucket plugin supports the following environment variables, which can override configuration file settings.

| Environment Variable            | Maps to Directive      | Description                                                                                              |
|----------------------------------|------------------------|----------------------------------------------------------------------------------------------------------|
| `SCANIO_BITBUCKET_USERNAME`     | `username`             | Bitbucket username for authentication. Overrides the `username` directive.                              |
| `SCANIO_BITBUCKET_TOKEN`        | `token`                | Bitbucket access token for authentication. Overrides the `token` directive.     |
| `SCANIO_BITBUCKET_SSH_KEY_PASSWORD`| `ssh_key_password`  | Password for the SSH key used in Bitbucket operations. Overrides the `ssh_key_password` directive.      |

## Usage Examples
### Command List
> [!IMPORTANT]  
> Currently, the plugin supports Bitbucket APIv1 only, which is still used for on-premises Bitbucket installations. Cloud installations, however, utilize Cloud REST API.

> [!NOTE]
> In Bitbucket, the term "namespace" refers to a project or user account that owns the repositories. 

#### Setup Prerequisites
**Authentication** <br>
To authenticate with Bitbucket, you must provide a valid access token. Scanio supports tokens of type Access Token.

More information about generating and managing Bitbucket personal access tokens can be found in the [official documentation](https://support.atlassian.com/bitbucket-cloud/docs/access-tokens/).

You can configure the token in one of two ways:
  - Directly in the [configuration file](../../config.yml):
    ```yaml
    bitbucket_plugin:
       username: "my-login"
       token: "my-token-example"
    ```
  - Using an environment variable:
    ```bash
    export SCANIO_BITBUCKET_USERNAME=my-login
    export SCANIO_BITBUCKET_TOKEN=my-token-example
    ```

**Access Permissions** <br>
The personal access token must have sufficient permissions to list repositories and access data. Required scopes include:
- Project permissions: Must be set to at least `read`. This grants access to repositories within the project.
- Repository permissions: Must be set to at least `read`. This allows access to repositories data.

For detailed information about possible configurations, please refer to [Configuration](configuration.md).

#### Validation
The Bitbucket plugin enforces the following validation rules:
- **Domain Validation**: Ensure that a domain argument is provided. This is mandatory for the plugin to operate.
- **Authentication Validation**: Both a valid access token and login must be supplied either through the configuration file or as environment variables. Without these, the operation cannot proceed.

#### Supported URL Types
The Bitbucket plugin supports the following URL types for the `list` command:

**Root VCS URL** <br>
Points to the root of the version control system.
```
https://bitbucket.com/
```

**Namespace URL** <br>
Points to a specific namespace.
```
https://bitbucket.com/projects/scanio/
https://bitbucket.com/scm/scanio/
ssh://git@bitbucket.com:7989/scanio/
```

**User Namespace URL** <br>
Points to a user's namespace.
```
https://bitbucket.com/users/shikari-ac
```

#### Actions
The Bitbucket plugin supports the following actions for the `list` command:

**List All Available Repositories in a VCS** <br>
Retrieve all repositories in the VCS by using either flags or a URL.

- Using Flags: Explicit control through the `--domain` flag.
    ```bash
    scanio list --vcs bitbucket --domain bitbucket.com -o /home/list_output.file
    ```

- Using a URL: Simplifies input by pointing directly to the VCS root.
    ```bash
    scanio list --vcs bitbucket -o /home/list_output.file https://bitbucket.com/
    ```

**List Repositories Within a Namespace** <br>
Retrieve repositories in a specific namespace by using flags or a URL.

- Using Flags: Explicit control through the `--domain` and `--namespace` flags.
    ```bash
    scanio list --vcs bitbucket --domain bitbucket.com --namespace scan-io-git -o /home/list_output.file
    ```

- Using a URL: Simplifies input by pointing directly to the namespace.
    ```bash
    scanio list --vcs bitbucket -o /home/list_output.file https://bitbucket.com/projects/scanio/
    scanio list --vcs bitbucket -o /home/list_output.file https://bitbucket.com/scm/scanio/ 
    scanio list --vcs bitbucket -o /home/list_output.file ssh://git@bitbucket.com:7989/scanio/
    ```

**List Repositories in a User Namespace** <br>

Retrieve repositories associated with a user namespace by using flags or a URL.

- Using Flags: Provides explicit control through the `--domain` and `--namespace` flags.
    ```bash
    scanio list --vcs bitbucket --domain bitbucket.com --namespace users/shikari-ac -o /home/list_output.file
    ```

- Using a URL: Simplifies input by pointing directly to the user namespace.
    ```bash
    scanio list --vcs bitbucket -o /home/list_output.file https://bitbucket.com/users/shikari-ac
    ```

### Command Fetch

> [!IMPORTANT]  
> Currently, the plugin supports Bitbucket APIv1 only, which is still used for on-premises Bitbucket installations. Cloud installations, however, utilize Cloud REST API.

> [!NOTE]
> In Bitbucket, the term "namespace" refers to a project or user account that owns the repositories. 

#### Setup Prerequisites
**Authentication**  
The Bitbucket plugin supports the following authentication methods:
- **HTTP Authentication**: Requires a personal access token.
- **SSH Agent Authentication**: Uses an existing SSH agent for authentication.
- **SSH Key Authentication**: Requires a path to a private SSH key.

*SSH Agent Authentication* <br>
Uses an existing SSH agent for authentication.

To use SSH agent authentication:
1. Add your private SSH key to the SSH agent:
   ```bash
   ssh-add /path/to/your/private/key
   ```
2. Ensure your public key is uploaded to your GitLab account.

*SSH Key Authentication* <br>
Uses a path to a private SSH key and password for the key if applicable. 

To use SSH key authentication:
1. Upload your public SSH key to your Bitbucket account.
2. Configure the key password:
   - Add them to the [configuration file](../../config.yml):
     ```yaml
     bitbucket_plugin:
       ssh_key_password: "" 
     ```
   - Or, use environment variables:
     ```bash
     export SCANIO_BITBUCKET_SSH_KEY_PASSWORD=my-password-example
     ```

*HTTP Authentication* <br>
To authenticate with Bitbucket, you must provide a valid access token. Scanio supports tokens of type Access Token.

1. Generate a personal access token in your GitLab account with the necessary scopes (`read` for Project and Repository).
   Refer to the [official documentation]((https://support.atlassian.com/bitbucket-cloud/docs/access-tokens/) for detailed instructions.
2. Configure the token:
   - Add it to the [configuration file](../../config.yml):
     ```yaml
     bitbucket_plugin:
       username: "my-login"
       token: "my-token-example"
     ```
   - Or, use an environment variable:
     ```bash
     export SCANIO_BITBUCKET_USERNAME=my-token-example
     export SCANIO_BITBUCKET_TOKEN=my-token-example
     ```

> [!TIP]
> Refer to [Configuration](configuration.md#bitbucket-plugin) for more details.

**Access Permissions** <br>
The personal access token must have sufficient permissions to fetch code. Required scopes include:
- Project permissions: Must be set to at least `read`. This grants access to repositories within the project.
- Repository permissions: Must be set to at least `read`. This allows access to repositories data.

For detailed information about possible configurations, please refer to [Configuration](configuration.md).

### Validation
The GitHub plugin includes additional validation to ensure correct operation:
- **URL Validation**: The URL for fetching argument must be provided. Without URL, the operation cannot proceed.
- **Authentication Type Validation**: The `--auth-type`, `-a` parameter must be provided and valid. Without authentication type, the operation cannot proceed.
- **Authentication Validation**: A valid access token and username is required for HTTP authentication type and SSH key and Password for SSH Key Authentication. Without proper authentication, the plugin will not function.
- **Consistency Check**: If the target folder already exists, the command verifies that its .git folder is intact before proceeding. This ensures the repository's integrity and supports restoration if files are missing or corrupted.

### Supported URL Types
The Bitbucket plugin supports multiple URL types for the `fetch` command:

**Repository URL** <br>
Points to a specific repository.
Example:
```
https://bitbucket.com/projects/SCANIO/repos/scan-io/browse # APIv1 HTTP URL
https://bitbucket.com/scm/SCANIO/scan-io.git # APIv1 HTTP-SCM URL
ssh://git@bitbucket.com:7989/SCANIO/scan-io.git # APIv1 SSH URL
https://bitbucket.com/users/scanio-bot/repos/scan-io/ # APIv1 HTTP URL User Repository
```

**URL with Specified Branch** <br>
Points to a specific repository with branch.
```
https://bitbucket.com/projects/SCANIO/repos/scan-io/browse?at=refs%2Fheads%2Ftest%2Ffeature # APIv1 HTTP URL
```

**Pull Request URL** <br>
Points to a particular pull request.
Example:
```
https:///bitbucket.com/projects/SCANIO/repos/scan-io/pull-requests/1/overview # APIv1 http URL
```

#### Diff artifacts (`--diff-lines`, `--diff-files`)

When `--diff-lines` or `--diff-files` is supplied for a pull-request URL, the Bitbucket plugin:

- clones the PR head inside the standard repository cache and computes the diff between the provider-reported base and head commits;
- materialises sparse diff files (only the added/modified lines) when `--diff-lines` is set, keeping unchanged lines blank for offset stability;
- copies the fully changed files when `--diff-files` is set so scanners that need full context can run without touching the rest of the checkout;
- copies dotfiles (for example `.gitignore`, scanner configuration files) into whichever diff folders are produced; and
- recreates the diff folders on every run so CI reruns do not accumulate stale artifacts.

The fetch response keeps `path` pointing to the repository checkout, sets `scope` to `diff-lines`, `diff-files`, or `diff` (when both flags are present), and populates the following `extras` keys:

| Key               | Description                                                 |
|-------------------|-------------------------------------------------------------|
| `diff_lines_root` | Absolute path to sparse diff artifacts (present with `--diff-lines`). |
| `diff_files_root` | Absolute path to full diff files (present with `--diff-files`). |
| `repo_root`       | Path to the fully cloned repository (also returned as `path`).|
| `base_sha`        | Provider base commit used for the diff (when available).     |
| `head_sha`        | Provider head commit used for the diff.                      |

Without diff flags, the plugin returns `scope: "full"` and `path`/`repo_root` both point to the repository checkout, matching the previous behaviour.

Original file (`config/app.env` before the PR):
```
1 DATABASE_URL=postgres://localhost/db
2 API_KEY=old-secret
3 CACHE_ENABLED=true
4 LOG_LEVEL=info
5 FEATURE_FLAG_X=false
```

New file (after PR changes):
```
1 DATABASE_URL=postgres://mysql/prod
2 API_KEY=new-rotated-secret
3 CACHE_ENABLED=true
4 LOG_LEVEL=debug
5 ERROR_REPORTING=enabled
6 FEATURE_FLAG_X=true
```

Unified diff between base and head:
```
diff --git a/config/app.env b/config/app.env
index 2b1e2d1..5ef9c42 100644
--- a/config/app.env
+++ b/config/app.env
@@
-DATABASE_URL=postgres://localhost/db
-API_KEY=old-secret
-CACHE_ENABLED=true
-LOG_LEVEL=info
-FEATURE_FLAG_X=false
+DATABASE_URL=postgres://mysql/prod
+API_KEY=new-rotated-secret
+CACHE_ENABLED=true
+LOG_LEVEL=debug
+ERROR_REPORTING=enabled
+FEATURE_FLAG_X=true
```

Sparse file written by diff-lines mode (the file stored under `<diff_lines_root>/config/app.env`; blank lines are intentional to preserve line numbers of unchanged lines).
```
1 DATABASE_URL=postgres://mysql/prod
2 API_KEY=new-rotated-secret
3 
4 LOG_LEVEL=debug
5 ERROR_REPORTING=enabled
6 FEATURE_FLAG_X=true
```

- Lines 1-2, 4-5, 7 hold the new values introduced by the diff.
- Line 3 was unchanged (`CACHE_ENABLED=true`), so it is left blank to keep offsets aligned.


### Actions
The Bitbucket plugin supports the following actions for the `fetch` command:

**Fetch a Specific Repository** <br>
This action retrieves the source code of a specified repository. 

The following examples demonstrate usage for various authentication methods:

```bash
# SSH Agent
scanio fetch --vcs bitbucket --auth-type ssh-agent https://bitbucket.com/projects/SCANIO/repos/scan-io/browse # APIv1 HTTP URL
scanio fetch --vcs bitbucket --auth-type ssh-agent https://bitbucket.com/scm/SCANIO/scan-io.git # APIv1 HTTP-SCM URL
scanio fetch --vcs bitbucket --auth-type ssh-agent ssh://git@bitbucket.com:7989/SCANIO/scan-io.git # APIv1 SSH URL

# SSH Key 
scanio fetch --vcs bitbucket --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 https://bitbucket.com/projects/SCANIO/repos/scan-io/browse # APIv1 HTTP URL
scanio fetch --vcs bitbucket --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 https://bitbucket.com/scm/SCANIO/scan-io.git # APIv1 HTTP-SCM URL
scanio fetch --vcs bitbucket --auth-typ ssh-key --ssh-key /Users/root/.ssh/id_ed25519 ssh://git@bitbucket.com:7989/SCANIO/scan-io.git # APIv1 SSH URL

# HTTP
scanio fetch --vcs bitbucket --auth-type http https://bitbucket.com/projects/SCANIO/repos/scan-io/browse # APIv1 HTTP URL
scanio fetch --vcs bitbucket --auth-type http https://bitbucket.com/scm/SCANIO/scan-io.git # APIv1 HTTP-SCM URL
scanio fetch --vcs bitbucket --auth-type http ssh://git@bitbucket.com:7989/SCANIO/scan-io.git # APIv1 SSH URL
```

For the following examples, we will use SSH-agent authentication, but all commands support all authentication types.

Fetching a specific repository supports specifying branches and particular commits:

```bash
## Branch
scanio fetch --vcs bitbucket --auth-type ssh-agent -b develop https://bitbucket.com/projects/SCANIO/repos/scan-io/browse # APIv1 HTTP URL
scanio fetch --vcs bitbucket --auth-type ssh-agent -b develop https://bitbucket.com/scm/SCANIO/scan-io.git # APIv1 HTTP-SCM URL
scanio fetch --vcs bitbucket --auth-type ssh-agent -b develop ssh://git@bitbucket.com:7989/SCANIO/scan-io.git # APIv1 SSH URL
scanio fetch --vcs bitbucket --auth-type ssh-agent https://bitbucket.com/projects/SCANIO/repos/scan-io/browse?at=refs%2Fheads%2Ftest%2Ffeature # Args derived from APIv1 HTTP URL

## Tag
scanio fetch --vcs bitbucket --auth-type ssh-agent -b tag https://bitbucket.com/projects/SCANIO/repos/scan-io/browse # APIv1 HTTP URL
scanio fetch --vcs bitbucket --auth-type ssh-agent -b tag https://bitbucket.com/scm/SCANIO/scan-io.git # APIv1 HTTP-SCM URL
scanio fetch --vcs bitbucket --auth-type ssh-agent -b tag ssh://git@bitbucket.com:7989/SCANIO/scan-io.git # APIv1 SSH URL

## Commit Hash
scanio fetch --vcs bitbucket --auth-type ssh-agent -b c0c9e9af80666d80e564881a5bdfa661c60e053e https://bitbucket.com/projects/SCANIO/repos/scan-io/browse # APIv1 HTTP URL
scanio fetch --vcs bitbucket --auth-type ssh-agent -b c0c9e9af80666d80e564881a5bdfa661c60e053e https://bitbucket.com/scm/SCANIO/scan-io.git # APIv1 HTTP-SCM URL
scanio fetch --vcs bitbucket --auth-type ssh-agent -b c0c9e9af80666d80e564881a5bdfa661c60e053e ssh://git@bitbucket.com:7989/SCANIO/scan-io.git # APIv1 SSH URL
```

Also, references avaliable via full reference name, for example:
```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent -b ref/heads/develop https://bitbucket.com/projects/SCANIO/repos/scan-io/browse # APIv1 HTTP URL
```

**Fetch a Specific Pull Request** <br>
This action allows you to fetch a specific pull request:

```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent https:///bitbucket.com/projects/SCANIO/repos/scan-io/pull-requests/1/overview
```

For fetching PRs are avaliable 3 methods:

*Branch* `--pr-mode branch`.

  This is the default mode for PR fetching — also known as the “feature branch” approach. It is the simplest and fastest (in the most cases) method, provided that the PR branch is accessible to the robot account within the same repository. At the plugin level, if no mode is explicitly specified with the `--pr-mode` flag, the source branch is resolved through the VCS API and then used by the underlying Git dependency.

In cases where the PR originates from a fork (for example, when the fork is private and not accessible to the robot account), this approach won’t work. In such scenarios, the next two modes are more appropriate.

*Special reference* `--pr-mode ref`.

Most VCS systems expose special references for pull requests, which point directly to the PR’s tip commit:
```
Bitbucket: refs/pull-requests/<ID>/from   (source tip)  
           refs/pull-requests/<ID>/merge  (synthetic merge)  
```
When cloning via references, the tool never uses the synthetic merge reference — only the head/from reference is fetched.

> [!WARNING] 
> Some VCS platforms use garbage collection, which may remove PR references after the PR is merged, making them unavailable later.

*Commit* `--pr-mode commit`.

A PR can also be fetched directly via its tip commit hash. In this case, the commit is checked out in detached mode, which comes with certain restrictions on local Git operations.

At the plugin level, the tip commit is resolved through the VCS API and then passed to the Git dependency for checkout.

**Fetch diff artifacts for a pull request** <br>
Add `--diff-lines` for sparse hunks, `--diff-files` for full file copies, or both to obtain every artifact in one run.

```bash
# Sparse diff lines
scanio fetch --vcs bitbucket --auth-type ssh-agent --diff-lines https:///bitbucket.com/projects/SCANIO/repos/scan-io/pull-requests/1/overview

# Full changed files
scanio fetch --vcs bitbucket --auth-type ssh-agent --diff-files https:///bitbucket.com/projects/SCANIO/repos/scan-io/pull-requests/1/overview

# Both outputs
scanio fetch --vcs bitbucket --auth-type ssh-agent --diff-lines --diff-files https:///bitbucket.com/projects/SCANIO/repos/scan-io/pull-requests/1/overview
```

**Approve or unapprove a pull request with head protection** <br>
Use `integration-vcs --action setStatusOfPR` with the optional `--require-head-sha` flag to ensure the PR head still matches the commit you scanned. If Bitbucket reports a different tip commit the command aborts, signalling that the review is stale.

```bash
scanio integration-vcs --vcs bitbucket --action setStatusOfPR --domain bitbucket.com \
  --status APPROVED --require-head-sha 08f766001d6ce714cb3619405a11ac69a942d8ca \ 
  https:///bitbucket.com/projects/SCANIO/repos/scan-io/pull-requests/1/overview
```

**Query pull request metadata and reviewer votes** <br>
`integration-vcs --action checkPR` outputs the Bitbucket pull request metadata that Scanio uses internally. The response now contains a `reviewers` array with each reviewer’s display name, email, approval state, and Bitbucket’s `lastReviewedCommit` hash when available.

```bash
scanio integration-vcs --vcs bitbucket --action checkPR --domain bitbucket.com \
  --namespace SCANIO --repository scan-io --pull-request-id 1
```

Use the `reviewers` array to ensure required reviewers approved (look for `APPROVED`) or to block automation when someone voted `REJECTED`. Review the [Integration-VCS command reference](cmd-integration-vcs.md) for schema details.

**Bulk Fetch from Input File** <br>
The `fetch` command seamlessly integrates with the `list` command by allowing users to use the output of the `list` command as input for fetching repositories. The `--input-file (-i)` option in the `fetch` command accepts a file generated by the `list` command. The format of the file aligns with the JSON structure documented in the [List Command Output Format](cmd-list.md#command-output-format).

> [!NOTE]  
> This feature is particularly useful for bulk repository fetching, enabling users to fetch multiple repositories in a single operation efficiently.

```bash
scanio fetch --vcs bitbucket --input-file /path/to/list_output.file --auth-type ssh-agent -j 5
```

**Optional Arguments** <br>
*Removing Extensions* <br>
To optimize the size of the fetched code and eliminate files that are generally excluded by security scanners, use the `--rm-ext` flag to specify file extensions for automatic removal.

```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent --rm-ext zip,tar.gz,log https://bitbucket.com/projects/SCANIO/repos/scan-io/browse
```

*Output Argument*  <br>
By default, the `fetch` command saves fetched repositories and pull requests to predefined directories:
- `{scanio_home_folder}/projects/<VCS_domain>/<namespace_name>/<repository_name>/` for fetched code.
- `{scanio_home_folder}/tmp/<VCS_domain>/<namespace_name>/<repository_name>/pr/<pr_id>` for fetcfetched pull requests.

If you want to customize the output location for fetched data, you can use the `--output` or `-o` flag. This flag allows you to specify a different directory for storing fetched repositories or pull requests.
```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent -o /path/to/repo_folder/ https://bitbucket.com/projects/SCANIO/repos/scan-io/browse
```

*Single Branch*  <br>
Fetch only the specified branch without history from other branches.

```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent --single-branch https://bitbucket.com/projects/SCANIO/repos/scan-io/browse
```

*Depth*  <br>
Create a shallow clone with a history truncated to the specified number of commits. Default: 0

```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent --depth 1 https://bitbucket.com/projects/SCANIO/repos/scan-io/browse
```

*Tags*  <br>
Fetch all tags from the repository.

```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent --tags https://bitbucket.com/projects/SCANIO/repos/scan-io/browse
```

*No Tags*  <br>
--no-tags - Do not fetch any tags from the repository.

```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent --no-tags https://bitbucket.com/projects/SCANIO/repos/scan-io/browse
```

*Auto Repair*  <br>
Added support for automatic repository repair when a fetch fails due to shallow-history or corrupted git history.

Behavior:
- On object not found or shallow-related errors, the client attempts to reclone the repository in place, using a safe temporary directory swap.
- This ensures resilience against shallow clones, force-pushes, or inconsistent remote states without requiring manual cleanup.

Why reclone instead of force-fetch? The underlying https://github.com/go-git/go-git/issues/1443 issue prevents force-fetch from being used in the scenario of shallow cloned repo.

```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent --auto-repair https://bitbucket.com/projects/SCANIO/repos/scan-io/browse
```

*Clean Workdir*  <br>
Introduced a clean working directory option (--clean-workdir) after checkout.

Behavior:
- Performs git reset --hardto align the worktree with the target commit/branch.
- Runs git clean -fdx equivalent, removing all untracked and ignored files.

Guarantees a deterministic and reproducible worktree, especially in CI/CD environments where leftover files can break builds or tests.

```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent --clean-workdir https://bitbucket.com/projects/SCANIO/repos/scan-io/browse
```

## Known Issues and Fixes
Below are some common errors users may encounter while using the Bitbucket plugin and suggested solutions to resolve them.

### ```ssh: handshake failed: knownhosts: key mismatch```
**Cause**<br>
This error occurs when there is a key mismatch in your SSH known hosts file due to incorrect port settings or missing configuration.

**Solution**<br>
Check your SSH configuration file (`~/.ssh/config`). If you are using a non-default SSH port (other than 22), you must explicitly specify the port for the host in the configuration:

```
Host git.example.com
   Hostname git.example.com
   Port 7989 
   IdentityFile ~/.ssh/id_ed25519
``` 

Alternatively, avoid using `.ssh/config` rules for this host, allowing the port to be identified automatically.

### ```ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain```
**Cause**<br>
This error suggests that SSH authentication methods are failing, often due to an incorrect or missing port specification in the SSH configuration.

**Solution**<br>
Ensure that the correct port is defined in your SSH configuration file for the respective host:
```
Host git.example.com
   Hostname git.example.com
   Port 7989 
   IdentityFile ~/.ssh/id_ed25519
``` 

Alternatively, avoid using `.ssh/config` rules for this host, allowing the port to be identified automatically.

### ```Error on Clone occurred: err="reference not found"```
**Cause**<br>
This error indicates that the specified branch does not exist in the remote repository.

**Solution**<br>
Verify and correct the branch name or repository URL in your fetch command.

### ```Error on Clone occurred: err="remote repository is empty"``` 
**Cause**<br>
This error appears when the default branch (e.g., `master` or `main`) in the remote repository is empty.

**Solution**<br>
Check the repository settings and ensure that the correct branch is specified in your fetch command.

### ```error creating SSH agent: "SSH agent requested but SSH_AUTH_SOCK not-specified"```
**Cause**<br>
This error occurs when the repository was initially cloned using SSH, and an attempt is made to fetch it using HTTP authentication. Git tries to use the existing SSH origin instead of switching to HTTP.

**Solution**<br>
To resolve this issue, either switch the authentication type to SSH or ensure consistency in authentication methods by avoiding mixed approaches within the same repository.

> [!IMPORTANT]  
> It is recommended not to mix different authentication methods within the same repository to avoid conflicts.

**Recommended Actions**<br>
Use SSH-based authentication for consistency:
```bash
scanio fetch --vcs bitbucket --auth-type ssh-agent https://bitbucket.com/projects/SCANIO/repos/scan-io/
```

If HTTP is required, update the repository's remote URL:
```bash
git remote set-url origin https://bitbucket.com/projects/SCANIO/repos/scan-io
```
