# GitLab Plugin

The GitLab plugin provides comprehensive support for interacting with GitLab version control systems. It offers a range of functionalities designed to streamline repository management and enhance CI/CD workflows, with a strong focus on security-related processes.

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

| API Version               | Supported | Note                                                                            |
|---------------------------|-----------|---------------------------------------------------------------------------------|
| GitLab REST API           |     ✅    | This is the current version used in GitLab Cloud installations. [Learn more](https://docs.gitlab.com/ee/api/rest/).|

## Supported Actions
| Action                                        | Command | Supported  |
|-----------------------------------------------|---------|------------|
| List all available repositories in a VCS      |   list  |     ✅     |
| List repositories within a namespace          |   list  |     ✅     |
| Filter repositories by programming language   |   list  |     ✅     |
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

## Configuration Refernces
### Configuration File
The GitLab plugin enables Scanio to interact with GitLab repositories for tasks such as listing projects, cloning repositories, and managing merge requests. This plugin uses the `gitlab_plugin` directive, which supports the following settings.

| Directive          | Default Value | Description                                                                                              |
|---------------------|---------------|----------------------------------------------------------------------------------------------------------|
| `username`         | `none`          | GitLab username for authentication. Optional, except when using HTTP for code fetching.                 |
| `token`            | `none`          | GitLab access token for authentication.                                                                |
| `ssh_key_password` | `none`          | Password for the SSH key used in GitLab operations (e.g., fetch command with `auth-type=ssh-key`).      |

### Environment Variables
The GitLab plugin supports the following environment variables, which can override configuration file settings.

| Environment Variable            | Maps to Directive      | Description                                                                                              |
|----------------------------------|------------------------|----------------------------------------------------------------------------------------------------------|
| `SCANIO_GITLAB_USERNAME`        | `username`             | GitLab username for authentication. Overrides the `username` directive.                                 |
| `SCANIO_GITLAB_TOKEN`           | `token`                | GitLab access token for authentication. Overrides the `token` directive.                                |
| `SCANIO_GITLAB_SSH_KEY_PASSWORD`| `ssh_key_password`     | Password for the SSH key used in GitLab operations. Overrides the `ssh_key_password` directive.         |

## Usage Examples
### Command List
> [!NOTE]
> In GitLab, the term "namespace" refers to a group or user account that owns the projects.

#### Setup Prerequisites
**Authentication** <br>
To authenticate with GitLab, you must provide a valid access token. Scanio supports tokens of type Personal Access Token.

More information about generating and managing GitLab personal access tokens can be found in the [official documentation](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html).

You can configure the token in one of two ways:
  - Directly in the [configuration file](../../config.yml):
    ```yaml
    gitlab_plugin:
      token: "my-token-example"
    ```
  - Using an environment variable:
    ```bash
    export SCANIO_GITLAB_TOKEN=my-token-example
    ```

**Access Permissions**  
Ensure the personal access token has sufficient permissions for listing repositories and accessing data. Required scopes include:
- `api`: Grants full access to the GitLab API, which is necessary for listing repositories.
- `read_api`: Allows read-only access to data within the GitLab instance, sufficient for most operations.

For detailed information about possible configurations, please refer to [Configuration](configuration.md).

#### Validation
The GitLab plugin enforces the following validation rules:
- **Domain Validation**: Ensure that a domain argument is provided. This is mandatory for the plugin to operate.
- **Authentication Validation**: A valid access token must be supplied either through the configuration file or as an environment variable. Without this, the operation cannot proceed.

#### Supported URL Types
The GitLab plugin supports the following URL types for the `list` command:

**Root VCS URL** <br>
Points to the root of the version control system.
```
https://gitlab.com/
```

**Namespace URL** <br>
Points to a specific namespace.
```
https://gitlab.com/testing_scanio/  
https://gitlab.com/testing_scanio/testingsubgroup/subgrouplevel2/  
```
Supports URLs with any number of subgroups.

> [!IMPORTANT]  
> By default, the parser interprets the last element of the URL as a project if the path contains two or more segments. This assumption is necessary because GitLab URLs do not inherently distinguish between namespaces and projects.

#### Actions
The GitLab plugin supports the following actions for the `list` command:

- **List All Available Repositories in a VCS**  
Retrieve all repositories in the VCS by using either flags or a URL.
- Using Flags: Explicit control through the `--domain` flag.
    ```bash
    scanio list --vcs gitlab --domain gitlab.com -o /home/list_output.file
    ```

- Using a URL: Simplifies input by pointing directly to the VCS root:
    ```bash
    scanio list --vcs gitlab -o /home/list_output.file https://gitlab.com/
    ```

**List Repositories Within a Namespace** <br>
Retrieve repositories available within a particular namespace by specifying either flags or a URL.
- Using Flags: Explicit control through the `--domain` and `--namespace` flags.
    ```bash
    scanio list --vcs gitlab --domain gitlab.com --namespace scan-io-git -o /home/list_output.file
    ```

- Using a URL: Simplifies input by pointing directly to the namespace.
    ```bash
    scanio list --vcs gitlab -o /home/list_output.file https://gitlab.com/testing_scanio/ 
    scanio list --vcs gitlab -o /home/list_output.file https://gitlab.com/testing_scanio/testingsubgroup/subgrouplevel2/   
    ```

**Filter Repositories by Programming Language** <br>
Narrow down repositories using a language filter.

- Using flags:
    ```bash
    scanio list --vcs gitlab --domain gitlab.com --namespace scan-io-git --language python -o /home/list_output.file
    ```

- Using URL:
    ```bash
    scanio list --vcs gitlab --language python -o /home/list_output.file https://gitlab.com/testing_scanio/ 
    ```

### Command Fetch
> [!NOTE]
> In GitLab, the term "namespace" refers to a group or user account that owns the projects.

#### Setup Prerequisites
**Authentication**  
The GitLab plugin supports the following authentication methods:
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
1. Upload your public SSH key to your GitLab account.
2. Configure the key password:
   - Add them to the [configuration file](../../config.yml):
     ```yaml
     gitlab_plugin:
       ssh_key_password: "" 
     ```
   - Or, use environment variables:
     ```bash
     export SCANIO_GITLAB_SSH_KEY_PASSWORD=my-password-example
     ```

*HTTP Authentication* <br>
To authenticate with GitLab, you must provide a valid access token. Scanio supports tokens of type Personal Access Token.

1. Generate a personal access token in your GitLab account with the necessary scopes (`api` and `read_api`).
   Refer to the [official documentation](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html) for detailed instructions.
2. Configure the token:
   - Add it to the [configuration file](../../config.yml):
     ```yaml
     gitlab_plugin:
       username: "my-login"
       token: "my-token-example"
     ```
   - Or, use an environment variable:
     ```bash
     export SCANIO_GITLAB_USERNAME=my-token-example
     export SCANIO_GITLAB_TOKEN=my-token-example
     ```

> [!TIP]
> Refer to [Configuration](configuration.md#gilab-plugin) for more details.

**Access Permissions** <br>
*HTTP Authentication* <br>
When using HTTP authentication, ensure that your access token has the required scopes and permissions to fetch the repository or repositories you want to access:
- Required Scopes for the Token:
  * `api`: Grants full access to the GitLab API.
  * `read_api`: Allows read-only access to data within the GitLab instance, sufficient for most operations.
- Repository Access:
  * Your GitLab account, associated with the token you use, must have at least read access to the repositories you plan to fetch.

*SSH Authentication (SSH Agent and SSH Key)* <br>
When using SSH-based authentication, ensure the following:
- Access to Repositories:
  * The SSH key must be added to your GitLab account and granted at least read access to the repositories you wish to fetch.
- Permissions:
  * If you are working with organizational repositories, verify that the SSH key has been authorized for access at the namespace level (if required).

#### Validation
The GitHub plugin includes additional validation to ensure correct operation:
- **URL Validation**: The URL for fetching argument must be provided. Without URL, the operation cannot proceed.
- **Authentication Type Validation**: The `--auth-type`, `-a` parameter must be provided and valid. Without authentication type, the operation cannot proceed.
- **Authentication Validation**: A valid access token and username is required for HTTP authentication type and SSH key and Password for SSH Key Authentication. Without proper authentication, the plugin will not function.
- **Consistency Check**: If the target folder already exists, the command verifies that its .git folder is intact before proceeding. This ensures the repository's integrity and supports restoration if files are missing or corrupted.

#### Supported URL Types
> [!NOTE]  
> GitLab URLs can include multiple subnamespaces or subgroups.

> [!IMPORTANT]  
> By default, the parser interprets the last element of the URL as a repository if the path contains two or more segments. This assumption is necessary because GitLab URLs do not inherently distinguish between namespaces and repositories.

**Repository URL** <br>
Points to a specific repository.
Example:
```
https://gitlab.com/testing_scanio/testing_scanio/  # HTTP type
https://gitlab.com/testing_scanio/testingsubgroup/subrouplevel2/projectlevel2/  # HTTP type
https://gitlab.com/testing_scanio/testing_scanio.git # HTTP type with .git
git@gitlab.com:testing_scanio/testing_scanio.git  # git@ type
```

**URL with Specified Branch** <br>
Points to a specific repository with branch.
```
https://gitlab.com/testing_scanio/testing_scanio/-/tree/test/feature  # HTTP type
```

**Pull Request URL** <br>
Points to a specific pull request.
Example:
```
https://gitlab.com/testing_scanio/testing_scanio/-/merge_requests/1  # HTTP type
https://gitlab.com/testing_scanio/testingsubgroup/subrouplevel2/projectlevel2/-/merge_requests/1  # HTTP type
```

#### Diff mode (`--diff`)

When `--diff` is provided for a merge-request URL, the GitLab plugin:

- clones the MR source branch/commit and produces a diff using GitLab’s `DiffRefs` (base and head SHAs) when available;
- writes only added/modified lines for each changed file into a cleaned `diff` directory under the MR temp path while leaving unchanged lines blank;
- copies dotfiles from the repository root into the diff directory so scanners retain configuration context; and
- recreates the diff directory on each invocation to ensure stale artifacts are removed in CI reruns.

The fetch response keeps `path` pointing to the repository checkout, sets `scope: "diff"`, and adds the following metadata:

| Key         | Description                                                                  |
|-------------|------------------------------------------------------------------------------|
| `diff_root` | Absolute path to the diff folder containing sparse files.                    |
| `repo_root` | Path to the repository checkout used when computing the diff (and returned as `path`). |
| `base_sha`  | Base commit SHA reported by GitLab (empty when the API omits it).            |
| `head_sha`  | Head commit SHA from `DiffRefs` or the MR SHA.                               |

Without `--diff`, the plugin returns `scope: "full"` and the `path`/`repo_root` both point to the repository checkout as in previous releases.

#### Actions
The GitLab plugin supports the following actions for the `fetch` command:

**Fetch a Specific Repository** <br>
This action retrieves the source code of a specified repository. 

The following examples demonstrate usage for various authentication methods:

```bash
# SSH Agent
scanio fetch --vcs gitlab --auth-type ssh-agent https://gitlab.com/testing_scanio/testing_scanio/ # HTTP URL
scanio fetch --vcs gitlab --auth-type ssh-agent git@gitlab.com:testing_scanio/testing_scanio.git # SSH URL


# SSH Key 
scanio fetch --vcs gitlab --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 https://gitlab.com/testing_scanio/testing_scanio/ # HTTP URL
scanio fetch --vcs gitlab --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 git@gitlab.com:testing_scanio/testing_scanio.git # SSH URL

# HTTP
scanio fetch --vcs gitlab --auth-type http https://gitlab.com/testing_scanio/testing_scanio/ # HTTP URL
scanio fetch --vcs gitlab --auth-type http git@gitlab.com:testing_scanio/testing_scanio.git # SSH URL
```

For the following examples, we will use SSH-agent authentication, but all commands support all authentication types.

Fetching a specific repository also supports specifying branches or commit hashes:

```bash
# Branch
scanio fetch --vcs gitlab --auth-type ssh-agent -b develop https://gitlab.com/testing_scanio/testing_scanio/ # HTTP URL
scanio fetch --vcs gitlab --auth-type ssh-agent -b develop git@gitlab.com:testing_scanio/testing_scanio.git # SSH URL
scanio fetch --vcs gitlab --auth-type ssh-agent https://gitlab.com/testing_scanio/testing_scanio/-/tree/test/feature  # Args derived from HTTP URL 

## Tag
scanio fetch --vcs gitlab --auth-type ssh-agent -b tag https://gitlab.com/testing_scanio/testing_scanio/ # HTTP URL
scanio fetch --vcs gitlab --auth-type ssh-agent -b tag git@gitlab.com:testing_scanio/testing_scanio.git # SSH URL

# Commit hash
scanio fetch --vcs gitlab --auth-type ssh-agent -b c0c9e9af80666d80e564881a5bdfa661c60e053e https://gitlab.com/testing_scanio/testing_scanio/ # HTTP URL
scanio fetch --vcs gitlab --auth-type ssh-agent -b c0c9e9af80666d80e564881a5bdfa661c60e053e git@gitlab.com:testing_scanio/testing_scanio.git # SSH URL
```

Also, references avaliable via full reference name, for example:
```bash
scanio fetch --vcs gitlab --auth-type ssh-agent -b ref/heads/develop https://gitlab.com/testing_scanio/testing_scanio/ # HTTP URL
```

**Fetch a Specific Pull Request** <br>
This action allows you to fetch a specific pull request:

```bash
scanio fetch --vcs gitlab --auth-type ssh-agent https://gitlab.com/testing_scanio/testing_scanio/-/merge_requests/1
```

For fetching PRs are avaliable 3 methods:

*Branch* `--pr-mode branch`.

  This is the default mode for PR fetching — also known as the “feature branch” approach. It is the simplest and fastest (in the most cases) method, provided that the PR branch is accessible to the robot account within the same repository. At the plugin level, if no mode is explicitly specified with the `--pr-mode` flag, the source branch is resolved through the VCS API and then used by the underlying Git dependency.

In cases where the PR originates from a fork (for example, when the fork is private and not accessible to the robot account), this approach won’t work. In such scenarios, the next two modes are more appropriate.

*Special reference* `--pr-mode ref`.

Most VCS systems expose special references for pull requests, which point directly to the PR’s tip commit:
```
GitLab:    refs/merge-requests/<ID>/head  
           refs/merge-requests/<ID>/merge  
```
When cloning via references, the tool never uses the synthetic merge reference — only the head/from reference is fetched.

> [!WARNING] 
> Some VCS platforms use garbage collection, which may remove PR references after the PR is merged, making them unavailable later.

*Commit* `--pr-mode commit`.

A PR can also be fetched directly via its tip commit hash. In this case, the commit is checked out in detached mode, which comes with certain restrictions on local Git operations.

At the plugin level, the tip commit is resolved through the VCS API and then passed to the Git dependency for checkout.

**Fetch only added/modified lines from a merge request** <br>
Use the `--diff` flag to persist only the new content and dotfiles required for scanning. The fetch response references the diff directory and provides base/head SHAs.

```bash
scanio fetch --vcs gitlab --auth-type ssh-agent --diff https://gitlab.com/testing_scanio/testing_scanio/-/merge_requests/1
```

**Bulk Fetch from Input File** <br>
The `fetch` command seamlessly integrates with the `list` command by allowing users to use the output of the `list` command as input for fetching repositories. The `--input-file (-i)` option in the `fetch` command accepts a file generated by the `list` command. The format of the file aligns with the JSON structure documented in the [List Command Output Format](cmd-list.md#command-output-format).

> [!NOTE]  
> This action is particularly useful for efficiently managing batch operations, especially in large-scale projects with multiple repositories.

```bash
scanio fetch --vcs gitlab --input-file /path/to/list_output.file --auth-type ssh-agent -j 5
```

**Optional Arguments** <br>
*Removing Extensions* <br>
To optimize the size of the fetched code and eliminate files that are generally excluded by security scanners, use the `--rm-ext` flag to specify file extensions for automatic removal.

```bash
scanio fetch --vcs gitlab --auth-type ssh-agent --rm-ext zip,tar.gz,log https://gitlab.com/testing_scanio/testing_scanio/
```

*Output Argument*  <br>
By default, the `fetch` command saves fetched repositories and pull requests to predefined directories:
- `{scanio_home_folder}/projects/<VCS_domain>/<namespace_name>/<repository_name>/` for fetched code.
- `{scanio_home_folder}/tmp/<VCS_domain>/<namespace_name>/<repository_name>/scanio-pr-tmp/<pr_id>` for fetcfetched pull requests.

If you want to customize the output location for fetched data, you can use the `--output` or `-o` flag. This flag allows you to specify a different directory for storing fetched repositories or pull requests.
```bash
scanio fetch --vcs gitlab --auth-type ssh-agent -o /path/to/repo_folder/https://gitlab.com/testing_scanio/testing_scanio/
```

*Single Branch*  <br>
Fetch only the specified branch without history from other branches.

```bash
scanio fetch --vcs gitlab --auth-type ssh-agent --single-branch https://gitlab.com/testing_scanio/testing_scanio/
```

*Depth*  <br>
Create a shallow clone with a history truncated to the specified number of commits. Default: 0

```bash
scanio fetch --vcs gitlab --auth-type ssh-agent --depth 1 https://gitlab.com/testing_scanio/testing_scanio/
```

*Tags*  <br>
Fetch all tags from the repository.

```bash
scanio fetch --vcs gitlab --auth-type ssh-agent --tags https://gitlab.com/testing_scanio/testing_scanio/
```

*No Tags*  <br>
--no-tags - Do not fetch any tags from the repository.

```bash
scanio fetch --vcs gitlab --auth-type ssh-agent --no-tags https://gitlab.com/testing_scanio/testing_scanio/
```

*Auto Repair*  <br>
Added support for automatic repository repair when a fetch fails due to shallow-history or corrupted git history.

Behavior:
- On object not found or shallow-related errors, the client attempts to reclone the repository in place, using a safe temporary directory swap.
- This ensures resilience against shallow clones, force-pushes, or inconsistent remote states without requiring manual cleanup.

Why reclone instead of force-fetch? The underlying https://github.com/go-git/go-git/issues/1443 issue prevents force-fetch from being used in the scenario of shallow cloned repo.

```bash
scanio fetch --vcs gitlab --auth-type ssh-agent --auto-repair https://gitlab.com/testing_scanio/testing_scanio/
```

*Clean Workdir*  <br>
Introduced a clean working directory option (--clean-workdir) after checkout.

Behavior:
- Performs git reset --hardto align the worktree with the target commit/branch.
- Runs git clean -fdx equivalent, removing all untracked and ignored files.

Guarantees a deterministic and reproducible worktree, especially in CI/CD environments where leftover files can break builds or tests.

```bash
scanio fetch --vcs gitlab --auth-type ssh-agent --clean-workdir https://gitlab.com/testing_scanio/testing_scanio/
```
