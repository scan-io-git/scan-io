# Commands
- Comand for listing repositories in a particular project.
```scanio list --vcs bitbucket --vcs-url bitbucket.com/rest --namespace AB -f output.file```
- Comand for resolving repositories in all projects.
```scanio list --vcs bitbucket --vcs-url bitbucket.com/rest -f output.file```

Listing output format is a file that is looks like:
    /<project_name>/<repo_name>
    /<project_name>/<repo_name>
    ...

- Comand for fetching particular repositories.
```scanio fetch --vcs bitbucket --vcs-url bitbucket.com --repos /ab/wcs```
--repos /<project_name>/<repo_name>
- Comand for fetching repositories from file.
```scanio fetch --vcs bitbucket --vcs-url git.acronis.com -f output.file -j 2```


# Environment for a BitBucket v1 API plugin
- BITBUCKET_USERNAME - yor usernmae in BitBucket.
- BITBUCKET_TOKEN - your token.
It may be plaint text password or personal access token form <your_bb_domain>/plugins/servlet/access-tokens/manage.

- BITBUCKET_SSH_PORT - port for git ssh operations. 
