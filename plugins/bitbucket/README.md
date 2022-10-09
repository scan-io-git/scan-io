# Commands
- Comand for listing repositories in a particular project.
```scanio list --vcs bitbucket --vcs-url bitbucket.com --namespace AB -f output.file```
- Comand for resolving repositories in all projects.
```scanio list --vcs bitbucket --vcs-url bitbucket.com -f output.file```

Listing output format is a file that is looks like:
    /<project_name>/<repo_name>
    /<project_name>/<repo_name>
    ...

- Comand for fetching particular repositories.
```scanio fetch --vcs bitbucket --vcs-url bitbucket.com --repos /ab/wcs```
--repos /<project_name>/<repo_name>
- Comand for fetching repositories from file.
```scanio fetch --vcs bitbucket --vcs-url git.acronis.com -f output.file -j 2```

# Errors
If you find error ```ssh: handshake failed: knownhosts: key mismatch ```
Check your .ssh/config. If you use not default 22 port for fetching and .ssh/config rules for this host, you have to determite a port too:
```
Host git.domain.com
   Hostname git.domain.com
   Port 7989 
   IdentityFile ~/.ssh/id_ed25519
``` 
Or just not using .ssh/config and port will be identifed automatically. 

```ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain```
Algorithm is the same - determite a port in .ssh/config for your host or don't use .ssh/config rules.


# Environment for a BitBucket v1 API plugin
- BITBUCKET_USERNAME - your usernmae in BitBucket.
- BITBUCKET_TOKEN - your token.
It may be a plain text password or personal access token from <your_bb_domain>/plugins/servlet/access-tokens/manage.

- BITBUCKET_SSH_KEY_PASSOWRD - your password for ssh
- BITBUCKET_SSH_PORT - port for git ssh operations. 


