# Commands
- Command for a listing repositories in a particular project.
```scanio list --vcs bitbucket --vcs-url bitbucket.com --namespace AB -f output.file```
- Command for a resolving repositories in all projects.
```scanio list --vcs bitbucket --vcs-url bitbucket.com -f output.file```

- Command for a fetching repositories from a file with ssh key authentifiction.
```scanio fetch --vcs bitbucket --vcs-url bitbucket.com --input-file output.file --auth-type ssh-key --ssh-key /Users/e.k/.ssh/id_ed25519 -j 1```
- Command for a fetching repositories from a file with ssh agent authentifiction.
```scanio fetch --vcs bitbucket --vcs-url bitbucket.com --input-file output.file --auth-type ssh-agent -j 1```
- Command for a fetching repositories from a file with http authentifiction.
```scanio fetch --vcs bitbucket --vcs-url bitbucket.com --input-file output.file --auth-typ http -j 1```

# Output
- The listing output format is a file that is looks like:
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
            "http_link": "https://git@domain.com/<project_name>/<repo_name>.git",
            "ssh_link": "ssh://git@git.acronis.com:7989/<project_name>/<repo_name>.git"
        }
    ],
    "status": "<status>",
    "message": "<err_message>"
}
```
- The fetching works without an output - only fetching repos on a disk. 

# Errors
- If you find an error ```ssh: handshake failed: knownhosts: key mismatch```
Check your .ssh/config. If you use not a default 22 port for fetching and .ssh/config rules for this host, you have to determite a port too:
```
Host git.domain.com
   Hostname git.domain.com
   Port 7989 
   IdentityFile ~/.ssh/id_ed25519
``` 
Or just not use .ssh/config and port will be identifed automatically. 

```ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain```
Algorithm is the same - determite a port in .ssh/config for your host or don't use .ssh/config rules.
- ```Error on Clone occured: err="reference not found"``` means that a branch in a remote repo doesn't exits.
- ```Error on Clone occured: err="remote repository is empty"``` means that a default branch in a remote repo (master/main) is empty.

# Environment for a BitBucket v1 API plugin
- BITBUCKET_USERNAME - your username in BitBucket. *mandatory
- BITBUCKET_TOKEN - your token. *mandatory
It may be a plain text password or a personal access token from <your_bb_domain>/plugins/servlet/access-tokens/manage.

- BITBUCKET_SSH_KEY_PASSOWRD - your password for ssh. Default is an empty value!
- BITBUCKET_SSH_PORT - port for git ssh operations. Default is 7989!