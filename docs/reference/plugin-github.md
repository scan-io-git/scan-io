# GitHub Plugin

The GitHub plugin provides comprehensive support for interacting with GitHub version control systems. It offers a range of functionalities designed to streamline repository management and enhance CI/CD workflows, with a strong focus on security-related processes.

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


## Supported Versions of API

| API Version                   | Supported | Note                                                                            |
|-------------------------------|-----------|---------------------------------------------------------------------------------|
| GitHub REST API - 2022-11-28  |     ✅    | This is the current version used in Github Cloud installations. [Learn more](https://docs.github.com/en/rest?apiVersion=2022-11-28).

## Supported Actions
| Action                                        | Command | Supported  |
|-----------------------------------------------|---------|------------|
| List all available repositories in a VCS      |   list  |     ✅     |
| List repositories within a namespace          |   list  |     ✅     |
| Filter repositories by programming language   |   list  |     ❌     |
| List repositories in a user namespace         |   list  |     ❌     |
| Fetch a specific repository                   |  fetch  |     ✅     |
| Fetch a specific pull request                 |  fetch  |     ✅     |
| Fetch repositories in bulk from an input file |  fetch  |     ✅     |

## Supported Authentication Types
| Authentication Type   | Command       | Supported  |
|-----------------------|---------------|------------|
| SSH key               |     fetch     |     ✅     |
| SSH agent             |     fetch     |     ✅     |
| HTTP                  |  list, fetch  |     ✅     |

## Configuration References
### Configuration File
The GitHub plugin enables Scanio to interact with GitHub repositories for various tasks such as listing projects, cloning code, and managing pull requests. This plugin uses the `github_plugin` directive, which supports the following settings.

| Directive          | Default Value | Description                                                                                              |
|---------------------|---------------|----------------------------------------------------------------------------------------------------------|
| `username`         | `none`          | GitHub username for authentication. Optional, except when using HTTP for code fetching.                 |
| `token`            | `none`          | GitHub access token for authentication.                                                                |
| `ssh_key_password` | `none`          | Password for the SSH key used in GitHub operations (e.g., fetch command with `auth-type=ssh-key`).      |

### Environment Variables
The GitHub plugin supports the following environment variables, which can override configuration file settings.

| Environment Variable            | Maps to Directive      | Description                                                                                              |
|----------------------------------|------------------------|----------------------------------------------------------------------------------------------------------|
| `SCANIO_GITHUB_USERNAME`        | `username`             | GitHub username for authentication. Overrides the `username` directive.                                 |
| `SCANIO_GITHUB_TOKEN`           | `token`                | GitHub access token for authentication. Overrides the `token` directive.                                |
| `SCANIO_GITHUB_SSH_KEY_PASSWORD`| `ssh_key_password`     | Password for the SSH key used in GitHub operations. Overrides the `ssh_key_password` directive.         |

## Usage Examples
### Command List
> [!NOTE]
> In GitHub, the term "namespace" refers to an organization or user account that owns the repositories.

#### Setup Prerequisites
**Authentication**  
To authenticate with GitHub, you must provide a valid access token. Supported token types:
- Fine-grained personal access tokens
- Personal access tokens (classic)

For more information, refer to the [GitHub Authentication Documentation](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens).

You can configure the token in one of two ways:
- [Configuration File](../../config.yml):
    ```yaml
    github_plugin:
      token: "my-token-example"
    ```

- Environment Variable:
    ```bash
    export SCANIO_GITHUB_TOKEN=my-token-example
    ```

> [!TIP]
> Refer to [Configuration](configuration.md#github-plugin) for more details.

**Access Permissions** <br>
The personal access token should have sufficient permissions to list repositories and access organizational or user-level data. Typically, the token requires
- `repo` scope for private repositories.
- `read:org` scope for organizational data.

#### Validation
The GitHub plugin includes additional validation to ensure correct operation:
- **Domain Validation**:  The `--domain` argument must be provided unless a valid URL is supplied. Without a domain or URL, the operation cannot proceed.
- **Authentication Validation**: A valid access token is required. This must be provided either through the configuration file or as an environment variable. Without proper authentication, the plugin will not function.

#### Supported URL Types
The GitHub plugin supports the following URL types for the `list` command:

**Root VCS URL** <br>
Points to the root of the VCS.
```
https://github.com/
```
**Namespace URL** <br>
Points to a specific namespace.
```
https://gitlab.com/scan-io-git/
```

#### Actions
The GitHub plugin supports the following actions for the `list` command:
**List All Available Repositories in a VCS** <br>
Retrieve repositories available within a VCS by specifying either flags or a URL.

- Using Flags: Explicit control through the `--domain` flag.
    ```bash
    scanio list --vcs github --domain github.com -o /home/list_output.file
    ```

- Using a URL: Simplifies input by pointing directly to the VCS root.
    ```bash
    scanio list --vcs github -o /home/list_output.file https://github.com/
    ```

**List Repositories Within a Namespace** <br>
Retrieve repositories available within a specific namespace by specifying either flags or a URL.

- Using Flags: Explicit control through the `--domain` and `--namespace` flags.
    ```bash
    scanio list --vcs github --domain github.com --namespace scan-io-git -o /home/list_output.file
    ```

- Using a URL: Simplifies input by pointing directly to the namespace.
    ```bash
    scanio list --vcs github -o /home/list_output.file https://gitlab.com/scan-io-git/
    ```
### Command Fetch
> [!NOTE]
> In GitHub, the term "namespace" refers to an organization or user account that owns the repositories.

#### Setup Prerequisites
**Authentication**  
The GitHub plugin supports three authentication methods.
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
2. Ensure your public key is uploaded to your GitHub account.

*SSH Key Authentication* <br>
Uses a path to a private SSH key and password for the key if applicable. 

To use SSH key authentication:
1. Upload your public SSH key to your GitHub account under "SSH and GPG keys" in account settings.
2. Configure the key password:
   - Add them to the [configuration file](../../config.yml):
     ```yaml
     github_plugin:
       ssh_key_password: "" 
     ```
   - Or, use environment variables:
     ```bash
     export SCANIO_GITHUB_SSH_KEY_PASSWORD=my-password-example
     ```

*HTTP Authentication* <br>
To authenticate with GitHub, you must provide a valid access token and username. Supported token types:
- Fine-grained personal access tokens
- Personal access tokens (classic)

1. Generate a personal access token in your GitHub account with the necessary scopes (`repo` and `read:org`).
   Refer to the [official documentation](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) for detailed instructions.
2. Configure the token:
   - Add it to the [configuration file](../../config.yml):
     ```yaml
     github_plugin:
       username: "my-login"
       token: "my-token-example"
     ```
   - Or, use an environment variable:
     ```bash
     export SCANIO_GITHUB_USERNAME=my-token-example
     export SCANIO_GITHUB_TOKEN=my-token-example
     ```

> [!TIP]
> Refer to [Configuration](configuration.md#github-plugin) for more details.

**Access Permissions** <br>
*HTTP Authentication* <br>
When using HTTP authentication, ensure that your access token has the required scopes and permissions to fetch the repository or repositories you want to access:
- Required Scopes for the Token:
  * `repo` scope for private repositories.
  * `read:org` scope for organizational data.
- Repository Access:
  * Your GitHub account, associated with the token you use, must have at least read access to the repositories you plan to fetch.

*SSH Authentication (SSH Agent and SSH Key)* <br>
When using SSH-based authentication, ensure the following:
- Access to Repositories:
  * The SSH key must be added to your GitHub account and granted at least read access to the repositories you wish to fetch.
- Permissions:
  * If you are working with organizational repositories, verify that the SSH key has been authorized for access at the organization level (if required).

#### Validation
The GitHub plugin includes additional validation to ensure correct operation:
- **URL Validation**: The URL for fetching argument must be provided. Without URL, the operation cannot proceed.
- **Authentication Type Validation**: The `--auth-type`, `-a` parameter must be provided and valid. Without authentication type, the operation cannot proceed.
- **Authentication Validation**: A valid access token and username is required for HTTP authentication type and SSH key and Password for SSH Key Authentication. Without proper authentication, the plugin will not function.
- **Consistency Check**: If the target folder already exists, the command verifies that its .git folder is intact before proceeding. This ensures the repository's integrity and supports restoration if files are missing or corrupted.

#### Supported URL Types
The GitHub plugin supports multiple URL types for the `fetch` command:

**Repository URL** <br>
Points to a specific repository.
Example:
```
https://github.com/scan-io-git/scan-io # HTTP type
https://github.com/scan-io-git/scan-io.git # HTTP type with .git
git@github.com:scan-io-git/scan-io.git # SSH type
```

**URL with Specified Branch** <br>
Points to a specific repository with branch.
```
https://github.com/scan-io-git/scan-io/tree/scanio_bot/test/feature  # HTTP type
```

**Pull Request URL** <br>
Points to a specific pull request.
Example:
```
https://github.com/scan-io-git/scan-io/pull/1 # HTTP type
```

#### Diff mode (`--diff`)

When `--diff` is supplied for a pull-request URL, the GitHub plugin:

- clones the PR head and computes the diff between the provider-reported base and head SHAs;
- writes only added/modified lines for files with statuses `added`, `modified`, or `renamed` into a `diff` directory beneath the PR temp path (unchanged lines are left blank);
- copies dotfiles (for example `.gitignore`, `.semgrep`) into the diff directory so scanner configuration is preserved; and
- recreates the diff directory on every run to avoid stale artifacts, which is especially useful in CI.

The resulting fetch response keeps `path` pointing to the repository checkout, sets `scope: "diff"`, and populates the following metadata in `extras`:

| Key         | Description                                                                  |
|-------------|------------------------------------------------------------------------------|
| `diff_root` | Absolute path to the sparse diff artifacts.                                  |
| `repo_root` | Path to the fully cloned repository (also returned as `path`).            |
| `base_sha`  | Base commit SHA returned by GitHub (when available).                          |
| `head_sha`  | Head commit SHA returned by GitHub.                                           |

Without `--diff`, the plugin returns `scope: "full"`; `path` and `extras.repo_root` both point to the repository root, matching the legacy behaviour.

#### Actions
The GitHub plugin supports the following actions for the `fetch` command:

**Fetch a Specific Repository** <br>

This action retrieves the source code of a specified repository. 

The following examples demonstrate usage for various authentication methods:

```bash
# SSH Agent
scanio fetch --vcs github --auth-type ssh-agent https://github.com/scan-io-git/scan-io # HTTP URL
scanio fetch --vcs github --auth-type ssh-agent git@github.com:scan-io-git/scan-io.git # SSH URL

# SSH Key 
scanio fetch --vcs github --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 https://github.com/scan-io-git/scan-io # HTTP URL
scanio fetch --vcs github --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 git@github.com:scan-io-git/scan-io.git # SSH URL

# HTTP
scanio fetch --vcs github --auth-type http https://github.com/scan-io-git/scan-io # HTTP URL
scanio fetch --vcs github --auth-type http git@github.com:scan-io-git/scan-io.git # SSH URL
```

For the following examples, we will use SSH-agent authentication, but all commands support all authentication types.

Fetching a specific repository also supports specifying branches or commit hashes:

```bash
## Branch
scanio fetch --vcs github --auth-type ssh-agent -b develop https://github.com/scan-io-git/scan-io # HTTP URL
scanio fetch --vcs github --auth-type ssh-agent -b develop git@github.com:scan-io-git/scan-io.git # SSH URL
scanio fetch --vcs github --auth-type ssh-agent https://github.com/scan-io-git/scan-io/tree/scanio_bot/test/feature # Args derived from HTTP URL 

## Tag
scanio fetch --vcs github --auth-type ssh-agent -b tag https://github.com/scan-io-git/scan-io # HTTP URL
scanio fetch --vcs github --auth-type ssh-agent -b tag git@github.com:scan-io-git/scan-io.git # SSH URL

## Commit Hash
scanio fetch --vcs github --auth-type ssh-agent -b c0c9e9af80666d80e564881a5bdfa661c60e053e https://github.com/scan-io-git/scan-io # HTTP URL
scanio fetch --vcs github --auth-type ssh-agent -b c0c9e9af80666d80e564881a5bdfa661c60e053e git@github.com:scan-io-git/scan-io.git # SSH URL
```

Also, references avaliable via full reference name, for example:
```bash
scanio fetch --vcs github --auth-type ssh-agent -b ref/heads/develop https://github.com/scan-io-git/scan-io # HTTP URL
```

**Fetch a Specific Pull Request** <br>
This action allows you to fetch a specific pull request:

```bash
scanio fetch --vcs github --auth-type ssh-agent https://github.com/scan-io-git/scan-io/pull/1
```

For fetching PRs are avaliable 3 methods:

*Branch* `--pr-mode branch`.

  This is the default mode for PR fetching — also known as the “feature branch” approach. It is the simplest and fastest (in the most cases) method, provided that the PR branch is accessible to the robot account within the same repository. At the plugin level, if no mode is explicitly specified with the `--pr-mode` flag, the source branch is resolved through the VCS API and then used by the underlying Git dependency.

In cases where the PR originates from a fork (for example, when the fork is private and not accessible to the robot account), this approach won’t work. In such scenarios, the next two modes are more appropriate.

*Special reference* `--pr-mode ref`.

Most VCS systems expose special references for pull requests, which point directly to the PR’s tip commit:
```
GitHub:    refs/pull/<ID>/head   (PR tip)  
           refs/pull/<ID>/merge  (synthetic merge)  
```
When cloning via references, the tool never uses the synthetic merge reference — only the head/from reference is fetched.

> [!WARNING] 
> Some VCS platforms use garbage collection, which may remove PR references after the PR is merged, making them unavailable later.

*Commit* `--pr-mode commit`.

A PR can also be fetched directly via its tip commit hash. In this case, the commit is checked out in detached mode, which comes with certain restrictions on local Git operations.

At the plugin level, the tip commit is resolved through the VCS API and then passed to the Git dependency for checkout.

**Fetch only added/modified lines from a pull request** <br>
Use the `--diff` flag to persist only the new content required for secrets or SAST scanning. The response references the diff folder and includes commit metadata.

```bash
scanio fetch --vcs github --auth-type ssh-agent --diff https://github.com/scan-io-git/scan-io/pull/1
```

**Bulk Fetch from Input File** <br>
The `fetch` command seamlessly integrates with the `list` command by allowing users to use the output of the `list` command as input for fetching repositories. The `--input-file (-i)` option in the `fetch` command accepts a file generated by the `list` command. The format of the file aligns with the JSON structure documented in the [List Command Output Format](cmd-list.md#command-output-format).

> [!NOTE]  
> This action is particularly useful for efficiently managing batch operations, especially in large-scale projects with multiple repositories.

```bash
scanio fetch --vcs github --input-file /path/to/list_output.file --auth-type ssh-agent -j 5
```

**Optional Arguments** <br>
*Removing Extensions* <br>
To optimize the size of the fetched code and eliminate files that are generally excluded by security scanners, use the `--rm-ext` flag to specify file extensions for automatic removal.

```bash
scanio fetch --vcs github --auth-type ssh-agent --rm-ext zip,tar.gz,log https://github.com/scan-io-git/scan-io
```

*Output Argument*  <br>
By default, the `fetch` command saves fetched repositories and pull requests to predefined directories:
- `{scanio_home_folder}/projects/<VCS_domain>/<namespace_name>/<repository_name>/` for fetched code.
- `{scanio_home_folder}/tmp/<VCS_domain>/<namespace_name>/<repository_name>/scanio-pr-tmp/<pr_id>` for fetcfetched pull requests.

If you want to customize the output location for fetched data, you can use the `--output` or `-o` flag. This flag allows you to specify a different directory for storing fetched repositories or pull requests.
```bash
scanio fetch --vcs github --auth-type ssh-agent -o /path/to/repo_folder/ https://github.com/scan-io-git/scan-io
```

*Single Branch*  <br>
Fetch only the specified branch without history from other branches.

```bash
scanio fetch --vcs github --auth-type ssh-agent --single-branch https://github.com/scan-io-git/scan-io
```

*Depth*  <br>
Create a shallow clone with a history truncated to the specified number of commits. Default: 0

```bash
scanio fetch --vcs github --auth-type ssh-agent --depth 1 https://github.com/scan-io-git/scan-io
```

*Tags*  <br>
Fetch all tags from the repository.

```bash
scanio fetch --vcs github --auth-type ssh-agent --tags https://github.com/scan-io-git/scan-io
```

*No Tags*  <br>
--no-tags - Do not fetch any tags from the repository.

```bash
scanio fetch --vcs github --auth-type ssh-agent --no-tags https://github.com/scan-io-git/scan-io
```

*Auto Repair*  <br>
Added support for automatic repository repair when a fetch fails due to shallow-history or corrupted git history.

Behavior:
- On object not found or shallow-related errors, the client attempts to reclone the repository in place, using a safe temporary directory swap.
- This ensures resilience against shallow clones, force-pushes, or inconsistent remote states without requiring manual cleanup.

Why reclone instead of force-fetch? The underlying https://github.com/go-git/go-git/issues/1443 issue prevents force-fetch from being used in the scenario of shallow cloned repo.

```bash
scanio fetch --vcs github --auth-type ssh-agent --auto-repair https://github.com/scan-io-git/scan-io
```

*Clean Workdir*  <br>
Introduced a clean working directory option (--clean-workdir) after checkout.

Behavior:
- Performs git reset --hardto align the worktree with the target commit/branch.
- Runs git clean -fdx equivalent, removing all untracked and ignored files.

Guarantees a deterministic and reproducible worktree, especially in CI/CD environments where leftover files can break builds or tests.

```bash
scanio fetch --vcs github --auth-type ssh-agent --clean-workdir https://github.com/scan-io-git/scan-io
```
