# Bitbucket plugin
The plugin implements work with functions ```list``` and ```fetch```:
* Listing all repositories in a VCS from the master/main branch.
* Listing repositories by a project in a VCS from the master/main branch.
* Fetching from an input file using an ssh-key/ssh-agent/HTTP authentification.<br><br>

This page is a short plugin description.<br>

You may find additional information in our articles:
- [scanio-list](../../docs/scanio-list.md).
- [scanio-fetch](../../docs/scanio-fetch.md).<br><br>

## Commands
* Listing all repositories in a VCS.<br>
```scanio list --vcs bitbucket --vcs-url example.com -f /Users/root/.scanio/output.file```
* Listing all repositories by a project in a VCS.<br>
```scanio list --vcs bitbucket --vcs-url example.com --namespace PROJECT -f /Users/root/.scanio/PROJECT.file```
* Listing all repositories in a VCS using URL.<br>
```scanio list --vcs bitbucket -f /Users/root/.scanio/PROJECT.file https://example.com/```
* Listing all repositories by a project using URL.<br>
```scanio list --vcs bitbucket -f /Users/root/.scanio/PROJECT.file https://example.com/projects/PROJECT/```
* Fetching from an input file using an ssh-key authentification.<br>
```scanio fetch --vcs bitbucket --input-file /Users/root/.scanio/output.file --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1```
* Fetching using an ssh-key authentification and URL that points a specific repository.<br>
```scanio fetch --vcs bitbucket --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1 https://example.com/projects/scanio_project/repos/scanio/browse```
* Fetching from an input file using an ssh-agent authentification.<br>
```scanio fetch --vcs bitbucket --input-file /Users/root/.scanio/output.file --auth-type ssh-agent -j 1```
* Fetching from an input file with an HTTP.<br>
```scanio fetch --vcs bitbucket --input-file /Users/root/.scanio/output.file --auth-typ http -j 1```<br><br>

### Supported URL formats
The application supports a few different formats of url:
* URL points to a VCS using Web UI format - ```https://example.com/```.<br>
&emsp;You could use the format with **list** command to list all repositories from your VCS.<br>
&emsp;```scanio list --vcs bitbucket -f /Users/root/.scanio/PROJECT.file https://example.com/```<br>
* URL points to a specific project using Web UI format - ```https://example.com/projects/<PROJECT_NAME>/```<br>
&emsp;You could use the format with **list** command to list all repositories from the project.<br>
&emsp;```scanio list --vcs bitbucket -f /Users/root/.scanio/PROJECT.file https://example.com/projects/scanio_project/```<br>
* URL points to a specific project and repository using Web UI format - ```https://example.com/projects/<PROJECT_NAME>/repos/<REPO_NAME>/browse```<br>
&emsp;You could use the format with **fetch** command to fetch a specific repository.<br>
&emsp;```scanio fetch --vcs bitbucket --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1 https://example.com/projects/scanio_project/repos/scanio/browse```<br>
* URL points to a specific project and repository using API format and ssh type - ```ssh://git@gexample.com:7989/<PROJECT_NAME>/<REPO_NAME>.git```<br>
&emsp;You could use the format with **fetch** command to fetch a specific repository.<br>
&emsp;You also can change the port using ssh scheme.<br>
&emsp;```scanio fetch --vcs bitbucket --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1 ssh://git@git.acronis.com:7989/scanio_project/scanio.git```<br>
* URL points to a specific project and repository using API format and http type - ```https://example.com/scm/<PROJECT_NAME>/<REPO_NAME>.git```<br>
&emsp;You could use the format with **fetch** command to fetch a specific repository.<br>
&emsp;```scanio fetch --vcs bitbucket --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1 https://git.acronis.com/scm/scanio_project/scanio.git```<br>

## Results of the command
### Output of a "list" command

As a result, the command prepares a JSON file:
```
{
    "args": {
        "Namespace": "<project_name>",
        "VCSURL": "<vcs_domain>"
    },
    "result": [
        {
            "namespace": "<project_name>",
            "repo_name": "<repo_name>",
            "http_link": "https://git@example.com/<project_name>/<repo_name>.git",
            "ssh_link": "ssh://git@git.example.com:7989/<project_name>/<repo_name>.git"
        }
    ],
    "status": "<status>",
    "message": "<err_message>"
}
```

### Output of a "fetch" command
The fetching works without an direct output.
The command saves results into a home directory ```~/.scanio/projects/+<VCSURL>+<Namespace>+<repo_name>```.<br><br>

## Possible errors
### ```ssh: handshake failed: knownhosts: key mismatch```
If you find the error check your .ssh/config. If you do use not a default 22 port for fetching and .ssh/config rules for this host, you have to determine a port too:
```
Host git.example.com
   Hostname git.example.com
   Port 7989 
   IdentityFile ~/.ssh/id_ed25519
``` 
Or just not use .ssh/config and the port will be identified automatically. <br><br>

### ```ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain```
The algorithm is the same - determine a port in .ssh/config for your host or don't use .ssh/config rules.<br><br>

### ```Error on Clone occurred: err="reference not found"``` 
It means that a branch in a remote repo doesn't exits. 
Try to fix the name of the branch or project.<br><br>

### ```Error on Clone occurred: err="remote repository is empty"``` 
It means that a default branch in a remote repo (master/main) is empty.
Try to fix the name of the branch or project.<br><br>

## Environment for a BitBucket v1 API plugin
* SCANIO_BITBUCKET_SSH_KEY_PASSOWRD - your password for ssh. The default is an empty value!
* SCANIO_BITBUCKET_USERNAME - your username in BitBucket.
* SCANIO_BITBUCKET_TOKEN - your Bitbucket token. 
  * It may be a plain text password or a personal access token from \<your_bb_domain\>/plugins/servlet/access-tokens/manage. <br><br>
