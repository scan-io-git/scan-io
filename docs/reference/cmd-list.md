# List Command
The `list` command in Scanio enables users to retrieve repository data from various version control systems (VCS). The table below outlines the supported platforms:

| VCS Platform       | Supported  |
|--------------------|------------|
| GitHub             | ✅         | 
| GitLab             | ✅         |
| Bitbucket APIv1    | ✅         |

The command offers features to filter repositories by domain, namespace, or programming language, depending on the selected VCS plugin.

## Table of Contents

- [Supported Actions](#supported-actions)
- [Syntax](#syntax)
- [Options](#options)
- [Core Validation](#core-validation)
- [Usage Examples](#usage-examples)
- [Command Output Format](#command-output-format)

## Supported Actions

| Action                                      | Supported Platforms          |
|---------------------------------------------|------------------------------|
| List all available repositories in a VCS    | GitHub, GitLab, Bitbucket    |
| List repositories within a namespace        | GitHub, GitLab, Bitbucket    |
| Filter repositories by programming language | GitLab                       |
| List repositories in a user namespace       | Bitbucket                    |


## Syntax
```bash
scanio list --vcs/-p PLUGIN_NAME --output/-o PATH [--language LANGUAGE] {--domain VCS_DOMAIN_NAME --namespace NAMESPACE | URL}
```

## Options

| Option           | Type   | Required   | Default Value | Description                                                                 |
| ---------------- | ------ | ---------- | ------------- | --------------------------------------------------------------------------- |
| `--domain`       | string | Conditional| `none`        | Domain name of the VCS (e.g., `github.com`). Required if not using a URL.   |
| `--help`         | flag   | No         | `false`       | Displays help for the `list` command.                                       |
| `--language`     | string | Optional   | `none`        | Filters repositories by language (GitLab only).                             |
| `--namespace`    | string | Conditional| `none`        | Name of the specific namespace, project, or organization. Optional to use with `--domain` if not using a URL. |
| `--output`, `-o` | string | Yes        | `none`        | Path to save the command results.                                           |
| `--vcs`, `-p`    | string | Yes        | `none`        | Specifies the VCS plugin to use (e.g., `bitbucket`, `gitlab`, `github`).    |


**Using URLs Instead of Flags** <br>
Instead of using the `--domain` and `--namespace` flags, you can specify a direct URL pointing to a namespace in your VCS. 

For detailed examples of supported URL formats per platform, refer to the plugin-specific examples.

### Core Validation
The `list` command includes multiple layers of validation to ensure proper execution:
- **Flag Requirements**: Ensures all required flags and parameters, as defined in the [Options](#options) table, are provided.
- **VCS Plugin Availability**: Validates the `--vcs/-p` flag against available plugins in the `plugins` directory. Only plugins with the type `vcs` are considered valid.
- **URL Parsing and Verification**: If a URL is provided, it is parsed using an internal [vcsurl dependency](../../pkg/shared/vcsurl/vcsurl.go). The core ensures the URL's validity and that it aligns with the expected structure for supported VCS platforms.

## Usage Examples
The following examples demonstrate the versatility of the `list` command with various plugins and configurations. These examples showcase its ability to list repositories, filter by namespace, apply language filters (where supported), and utilize direct URLs for streamlined operations.

Refer to plugin-specific documentation for detailed examples and additional requirements of the command usage:
- [GitHub Plugin - Command List](plugin-github.md#command-list)
- [GitLab Plugin - Command List](plugin-gitlab.md#command-list)
- [Bitbucket Plugin - Command List](plugin-bitbucket.md#command-list)

## Command Output Format
The `list` command generates a JSON file as output, a path to save the output is taken from `--output/-o`. This file serves as both the result of the command and an input for other commands in Scanio's workflow.

```json
{
    "launches": [
        {
            "args": {
                "repo_param": {
                    "domain": "<domain_name>",
                    "namespace": "<namespace_name>",
                    "repository": "<repository_name>",
                    "http_link": "<http_link>",
                    "ssh_link": "<ssh_link>"
                },
                "action": "list",
                "language": "<language>"
            },
            "result": [
                {
                    "namespace": "<namespace_name>",
                    "repository": "<repository_name>",
                    "http_link": "<http_link>",
                    "ssh_link": "<ssh_link>"
                },                 
                {
                    "namespace": "<namespace_name>",
                    "repository": "<repository_name>",
                    "http_link": "<http_link>",
                    "ssh_link": "<ssh_link>"
                }
            ],
            "status": "<status>",
            "message": "<error_message>"
        }
    ]
}
```

### Key Fields
| Field       | Description                                                                 |
|-------------|-----------------------------------------------------------------------------|
| `args`      | Dictionary containing the arguments used to execute the command.            |
| `result`    | List of dictionaries representing the actual command results.               |
| `status`    | String indicating the final status of the command (e.g., `OK`, `FAILED`).   |
| `message`   | String containing error messages or `stderr` output if the status is not `OK`.|

### Fields in the `args` Object

| Field       | Description                                                                 |
|-------------|-----------------------------------------------------------------------------|
| `repo_param`| Contains repository-specific parameters, including domain and namespace.    |
| `action`    | Specifies the action performed by the command (list).                       |
| `language`  | Specifies the language filter applied (if any).                             |

### Fields in the `repo_param` Object

| Field       | Description                                                                 |
|-------------|-----------------------------------------------------------------------------|
| `domain`    | The domain name of the VCS (e.g., github.com).                              |
| `namespace` | The namespace, project, or organization name in the VCS.                    |
| `repository`| The name of the repository in the VCS.                                      |
| `http_link` | The `https://` URL used for fetching the repository from the VCS.           |
| `ssh_link`  | The `ssh://` URL used for fetching the repository from the VCS.           |

### Fields in the `result` List
| Field       | Description                                                                 |
|-------------|-----------------------------------------------------------------------------|
| `namespace` | Name of the project, organization, or namespace in the VCS.                 |
| `repository`| Name of the repository in the VCS.                                          |
| `http_link` | The `https://` URL used for fetching the repository from the VCS.           |
| `ssh_link`  | The `ssh://` URL used for fetching the repository from the VCS.           |
