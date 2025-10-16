# Fetch Command
The `fetch` command retrieves the source code of repositories from a specified version control system (VCS) while ensuring consistency. If local files are deleted or corrupted, the command can restore them, provided the .git folder remains intact and uncorrupted.

This command supports operations at different levels, including individual repositories and pull requests. Fetch operations are designed with consistency checks and optional concurrent job support.

| VCS Platform       | Supported  |
|--------------------|------------|
| GitHub             |     ✅     |
| GitLab             |     ✅     |
| Bitbucket          |     ✅     |

## Table of Contents

- [Supported Actions](#supported-actions)
- [Supported Authentication Types](#supported-authentication-types)
- [Syntax](#syntax)
- [Options](#options)
- [Core Validation](#core-validation)
- [Usage Examples](#usage-examples)
- [Command Output Format](#command-output-format)

## Supported Actions
| Action                                        | Supported Platforms          |
|-----------------------------------------------|------------------------------|
| Fetch a specific repository                   | GitHub, GitLab, Bitbucket    |
| Fetch a specific pull request                 | GitHub, GitLab, Bitbucket    |
| Fetch repositories in bulk from an input file | GitHub, GitLab, Bitbucket    |

## Supported Authentication Types
| Authentication Type                         | Supported Platforms          |
|---------------------------------------------|------------------------------|
| SSH key                                     | GitHub, GitLab, Bitbucket    |
| SSH agent                                   | GitHub, GitLab, Bitbucket    |
| HTTP                                        | GitHub, GitLab, Bitbucket    |

## Syntax
```bash
scanio fetch --vcs/-p PLUGIN_NAME --auth-type/-a AUTH_TYPE [--ssh-key/-k PATH] [--output/-o PATH] [--rm-ext LIST_OF_EXTENSIONS] [-j THREADS_NUMBER, default=1] {--input-file/-i PATH | [-b BRANCH/HASH] URL}
```

### Options
| Option           | Type    | Required    | Default Value                                                | Description                                                                 |
|-------------------|---------|-------------|--------------------------------------------------------------|-----------------------------------------------------------------------------|
| `--auth-type`, `-a` | string  | Yes         | `none`                                                       | Type of authentication to use (e.g., `http`, `ssh-agent`, `ssh-key`).        |
| `--branch`, `-b` | string  | Conditional | `main` or `master`                                           | The specific branch or commit hash to fetch.                                |
| `--help`, `-h`   | flag    | No          | `false`                                                      | Displays help for the `fetch` command.                                      |
| `--input-file`, `-i` | string | Conditional | `none`                                                       | Path to a file in [Scanio list command](cmd-list.md#command-output-format) format containing repositories to fetch.            |
| `--output`, `-o` | string  | No          | `{scanio_home_folder}/projects/<VCS_domain>/<namespace_name>/<repository_name>`      | Path to save the fetched repository code.         |
| `--rm-ext`       | strings | No          | `[csv,png,ipynb,txt,md,mp4,zip,gif,  gz,jpg,jpeg,cache,tar,svg,bin,lock,exe]` | Comma-separated list of file extensions to remove after fetching.           |
| `--ssh-key`, `-k` | string  | Conditional | `none`                                                       | Path to the SSH key to use (if `auth-type` is `ssh-key`).                    |
| `--threads`, `-j`| int     | No          | `1`                                                          | Number of concurrent threads to use for parallel fetching.                   |
| `--vcs`, `-p`    | string  | Yes         | `none`                                                       | Specifies the VCS plugin to use (e.g., `bitbucket`, `gitlab`, `github`).     |
| `--pr-mode`   | string  | No         | `branch`                                                       | Pull request fetch strategy. References are resolved automatically via the VCS provider API — users don’t need to specify refs manually. Possible values: branch (fetch the PR’s source branch), ref (fetch the provider’s PR ref, e.g., GitHub refs/pull/<id>/head), or commit (fetch the PR head’s latest commit in detached mode). Ignored if URL doesn't contain PR ID.     |
| `--diff`      | bool    | No         | `false`                                                        | For pull-request URLs, persist only added/modified lines (plus dotfiles) into a dedicated diff folder instead of copying the entire repository tree. The fetch response includes metadata pointing at the diff artifacts. |
| `--single-branch`   | bool  | No         | `false`                                                       | Fetch only the specified branch, without history from other branches.     |
| `--depth`   | int  | No         | `0`                                                       | Create a shallow clone with history truncated to n commits. `0` = full history, `1` = shallowest. |
| `--tags`   | bool  | No         | `false`                                                       | Fetch all tags from the repository. If neither `--tags` nor `--no-tags` is set, tag-following is used by default.    |
| `--no-tags`   | bool  | No         | `false`                                                       | Do not fetch any tags from the repository. If neither `--tags` nor `--no-tags` is set, tag-following is used by default.     |
| `--auto-repair`   | bool  | No         | `false`                                                       | Automatically repair shallow or corrupted repositories by forcing a refetch and recloning if necessary.     |
| `-clean-workdir`   | bool  | No         | `false`                                                       | Reset the working tree to HEAD and remove untracked files and directories. Equivalent to `git reset --hard` + `git clean -fdx`.     |


**Using List Command Output as Input**<br>
The `fetch` command integrates seamlessly with the list command, allowing users to input a file generated by the `list` command for bulk repository fetching. The `--input-file (-i)` flag accepts files in the format outlined in the [List Command Output Format](cmd-list.md#command-output-format).

> [!NOTE]  
> The feature is particularly useful for bulk repository fetching, enabling users to fetch multiple repositories in a single operation efficiently.

```bash
scanio fetch --vcs github --input-file /path/to/list_output.file --auth-type ssh-agent -j 5
```

**Using URLs**<br>
The `fetch` command supports the use of URLs to specify repositories or pull requests. These URLs must point directly to repositories, and their format may vary depending on the VCS plugin being used.

For detailed examples of supported URL formats per platform, refer to the plugin-specific examples.

### Core Validation
The `fetch` command includes several validation layers to ensure robust execution and accurate results:
- **Flag Requirements**: Ensures all required flags and parameters, as defined in the [Options](#options) table, are provided.
- **VCS Plugin Availability**: Validates the `--vcs/-p` flag against available plugins in the `plugins` directory. Only plugins with the type `vcs` are considered valid.
- **Authentication Validation**: Ensures the specified `--auth-type` is valid. If `ssh-key` is selected, the presence of a valid `--ssh-key` path and key is required.
- **Input Validation**: Confirms that either an `--input-file` or a valid URL is provided. Both cannot be omitted simultaneously.
- **URL Parsing and Verification**: If a URL is provided, it is parsed using the internal [vcsurl dependency](../../pkg/shared/vcsurl/vcsurl.go). The core ensures the URL's validity and that it aligns with the expected structure for supported VCS platforms.

## Usage Examples
The following examples demonstrate how to use the `fetch` command for different plugins. Each example covers specific use cases, such as fetching repositories, pull requests, using authentication methods, and managing parallel jobs.

Refer to plugin-specific documentation for detailed examples and additional requirements of the command usage:
- [GitHub Plugin: Command Fetch](plugin-github.md#command-fetch)
- [GitLab Plugin: Command Fetch](plugin-gitlab.md#command-fetch)
- [Bitbucket Plugin: Command Fetch](plugin-bitbucket.md#command-fetch)

## Command Output Format
The `fetch` command generates a JSON file as output, capturing detailed information about the execution process, arguments, and results.

```json
{
  "launches": [
    {
      "args": {
        "clone_url": "<clone_url>",
        "branch": "<branch_name/commit_hash>",
        "auth_type": "<auth_type>",
        "ssh_key": "<path_to_ssh_key>",
        "target_folder": "<path_to_folder_for_code>",
        "fetch_mode": "<fetch_mode>",
        "repo_param": {
          "domain": "<domain_name>",
          "namespace": "<namespace_name>",
          "repository": "<repository_name>",
          "pull_request_id": "<pr_id>",
          "http_link": "<http_link>",
          "ssh_link": "<ssh_link>"
        },
        "depth": "<depth>",
        "single_branch": "<single_branch_bool>",
        "tag_mode": "<tag_mode>",
      },
      "result": {
        "path": "<path_to_folder_with_saved_code>"
      },
      "status": "<status>",
      "message": "<error_message>"
    }
  ]
}
```

### Key Fields
| Field       | Description                                                                   |
|-------------|-------------------------------------------------------------------------------|
| `args`      | Dictionary containing the arguments used to execute the command.              |
| `result`    | List of dictionaries representing the actual command results.                 |
| `status`    | String indicating the final status of the command (e.g., `OK`, `FAILED`).     |
| `message`   | String containing error messages or `stderr` output if the status is not `OK`.|

### Fields in the `args` Object

| Field       | Description                                                                 |
|-------------|-----------------------------------------------------------------------------|
| `clone_url`| The URL used to clone the repository.                                        |
| `branch`| Specifies the branch name or commit hash to fetch.                              |
| `auth_type`| Specifies the authentication method used.                                    |
| `ssh_key`| Path to the SSH key file if auth_type is ssh-key.                              |
| `target_folder`| Path to the folder where the repository code will be saved.              |
| `fetch_mode`| The fetch mode (`basic`, `pull-branch`, `pull-ref`, `pull-commit`).         |
| `repo_param`| Contains repository-specific parameters, including domain and namespace.    |
| `depth`| Actuall depth during cloning/fetching by git dependency.                         |
| `single_branch`| Fetch only the specified branch status.                                  |
| `tag_mode`| Tag mode used during cloning by git dependency (all, no, or following tags)   |


### Fields in the `repo_param` Object

| Field       | Description                                                                 |
|-------------|-----------------------------------------------------------------------------|
| `domain`    | The domain name of the VCS (e.g., github.com).                              |
| `namespace` | The namespace, project, or organization name in the VCS.                    |
| `repository`| The name of the repository in the VCS.                                      |
| `pull_request_id`| The ID of the pull request (if fetching a PR).                         |
| `http_link` | The `https://` URL used for fetching the repository from the VCS.           |
| `ssh_link`  | The `ssh://` URL used for fetching the repository from the VCS.             |

### Fields in the `result` List
| Field       | Description                                                                 |
|-------------|-----------------------------------------------------------------------------|
| `path`      | Path to the repository checkout on disk (always the repo root). When `--diff` is enabled use `extras.diff_root` to locate the sparse diff artifacts. |
| `scope`     | Represents the fetch scope (`full` or `diff`). Mirrors the CLI flag and allows downstream automation to branch logic. |
| `extras`    | Key/value metadata returned by the VCS plugin. For diff fetches this includes `diff_root`, `repo_root`, and the `base_sha`/`head_sha` pair when available. |
