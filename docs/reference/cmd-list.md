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
   - [GitHub Plugin](#github-plugin)
     - [Setup Prerequisites](#setup-prerequisites)
     - [Validation](#validation)
     - [Supported URL Types](#supported-url-types)
     - [Actions](#actions)
   - [GitLab Plugin](#gitlab-plugin)
     - [Setup Prerequisites](#setup-prerequisites-1)
     - [Validation](#validation-1)
     - [Supported URL Types](#supported-url-types-1)
     - [Actions](#actions-1)
   - [Bitbucket Plugin](#bitbucket-plugin)
     - [Setup Prerequisites](#setup-prerequisites-2)
     - [Validation](#validation-2)
     - [Supported URL Types](#supported-url-types-2)
     - [Actions](#actions-2)
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

### Plugin-Specific Documentation
Refer to the sections below for detailed plugin-specific examples and additional requirements:
- [GitHub Plugin](#github-plugin)
- [GitLab Plugin](#gitlab-plugin)
- [Bitbucket Plugin](#bitbucket-plugin)

### GitHub Plugin
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

### GitLab Plugin
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

### Bitbucket Plugin
> [!IMPORTANT]  
> Currently, the plugin supports Bitbucket APIv1 only, which is still used for on-premises Bitbucket installations. Cloud installations, however, utilize APIv2.

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
