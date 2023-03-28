# Scanio List Command
The main command's function is to list repositories from a version control system.<br><br>

The command can list repositories:
- In the whole VCS. The results will be a file with all repositories in your VCS.
- By projects or namespaces. The results will be a file with all repositories in a particular namespace.
<br><br>

Covered VCSs:
- Bitbucket API v1.
- Github.
- Gitlab.<br>

|    |Bitbucket|Gitlab|Github|
|----|-----|---|---|
|List in all VCS|Supported|Not Supported|?|
|List by all VCS from URL|Supported|Not Supported|?|
|List by a project from args|Supported|Supported|?|
|List by a project from URL|Supported|Not Supported|?|
|List an only one repository|Not Supported|Not Supported|Not Supported|
|Public repositories|Supported|Supported|Supported|
|Private repositories|Supported| ?|Not Supported|
|Users repositories|Not supported|Not supported|Not supported| 

## Result of the Command
As a result, the command prepares a JSON file:
```
{
    "args": {
        "Namespace": "<project_name>,
        "VCSURL": "<vcs_domain>"
    },
    "result": [
        {
            "namespace": "<project_name>",
            "repo_name": "<repo_name>",
            "http_link": "https://git@<vcs_domain>/<project_name>/<repo_name>.git",
            "ssh_link": "ssh://git@git.<vcs_domain>:7989/<project_name>/<repo_name>.git"
        }
    ],
    "status": "<status>",
    "message": "<err_message>"
}
```

Where:
* "args" is a dictionary with used arguments. It needs to for debugging reasons.
* "results" is a list of dictionaries that consist of an actual result of the command's work. 
* "status" is a string with the final status of the command. Eg. "OK", "FAILED". It needs to for debugging reasons.
* "message" is a string with a stderr output if the status is not "OK". It needs to for debugging reasons.<br><br>

The dictionaries in a "result" list consist of:
* "namespace" is a name of a project in your VCS.
* "repo_name" is a name of a repository in your VCS. 
* "http_link" is a link with an `https://` scheme from a VCS API for fetching.
* "ssh_link" is a link with an `ssh://` scheme from a VCS API for fetching.<br>

The path in ```http_link/ssh_link``` might be different. It depends on the VCS due to each VCS has a different tree of projects and repositories. <br><br>

This generic output is used as input for other commands in case of no manual interaction with the tool.<br>

## Args of the Command
* "vcs" is the plugin name of the VCS used. Eg. bitbucket, gitlab, github, etc.
* "vcs-url" is an URL to a root of the VCS API. Eg. github.com.
* "output" or "f" is a path to an output file.
* "namespace" is the name of a specific namespace. Namespace for Gitlab is an organization, for Bitbucket_v1 is a project.
* "language" or "l" helps to collect only projects that have code in a specified language. It works only for Gitlab.<br><br>

Instead of using **vcs-url** and **namespace** flags you could use a specific **URL** that points to a particular namespace from your VCS (check the [link](#listing-all-repositories-by-a-project-using-url)) and points to VCS (check the [link](#listing-all-repositories-in-a-vcs-using-url)).

## Using Scenarios 
When developing, we aimed at the fact that the program will be used primarily for automation purposes but you still able to use it manually from CLI.

### Listing All Repositories in a VCS
This scenario needs if you would like to list all repositories from your VCS.

#### Bitbucket
```scanio list --vcs bitbucket --vcs-url example.com -f /Users/root/.scanio/output.file```

#### Github
At the moment the plugin can't list a all VCS.

#### Gitlab
*TODO*

### Listing All Repositories in a VCS Using URL
This scenario needs if you would like to list all repositories from your VCS by using URL.

#### Bitbucket
```scanio list --vcs bitbucket -f /Users/root/.scanio/PROJECT.file https://example.com/```<br>
You can find additional information about URL formats [here](../plugins/bitbucket/README.md#supported-url-formats)

#### Github
*Not supported*

#### Gitlab
*Not supported*

### Listing Repositories by a Project in a VCS
This scenario needs if you would like to list repositories on a specified project/namespace. 

#### Bitbucket
```scanio list --vcs bitbucket --vcs-url example.com --namespace PROJECT -f /Users/root/.scanio/PROJECT.file```

#### Github
```scanio list --vcs github --vcs-url example.com --namespace PROJECT -f /Users/root/.scanio/PROJECT.file```

#### Gitlab
*TODO*

### Listing All Repositories by a Project Using URL
This scenario needs if you would like to list repositories on a specified project/namespace by using URL. 

#### Bitbucket
```scanio list --vcs bitbucket -f /Users/root/.scanio/PROJECT.file https://example.com/projects/PROJECT/```<br>
You can find additional information about URL formats [here](../plugins/bitbucket/README.md#supported-url-formats)

#### Github
*Not supported*

#### Gitlab
*Not supported*

## Authentification
If your VCS requires an authentification or your ```project/namespace/repository``` is private you will have to auth factor for an authentification.

### Bitbucket
For Bitbucket API v1 you need to use a few environment variables:
* SCANIO_BITBUCKET_USERNAME - your username in a VCS.
* SCANIO_BITBUCKET_TOKEN - token for authentification.
   * It may be a plain text password or a personal access token from ```<your_bb_domain>/plugins/servlet/access-tokens/manage```.

### Github
The plugin can list only public repositories.

### Gitlab
For Gitlab you need to use an environment variable:
- GITLAB_TOKEN - token for an authentification.
