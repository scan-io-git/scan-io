# Current document purpose
This guide is aimed to help you unserstand how scanio works and setup you scans in k8s cluster as fast as possible.

## Step 1. Locally
At the first step we advice you to work with scanio locally.  
We are going to cover few commands on this step: how to get list of repositories and fetch them on local host.
1. Generate a personal access token: https://gitlab.com/-/profile/personal_access_tokens
2. Set env var: `export GITLAB_TOKEN=[...redacted...]`
3. Get list of projects in specific group (namespace): `scanio list --vcs gitlab --vcs-url gitlab.com --namespace demo-group --output /tmp/demo-group-projects.json`
4. If you want to get all projects for all groups (at will take much more time): `scanio list --vcs gitlab --vcs-url gitlab.com --output /tmp/demo-projects.json`
5. You can also filter by interested language (this will not increase the time comparing with previous command): `scanio list --vcs gitlab --vcs-url gitlab.com --output /tmp/demo-projects.json -l python`
As a result scanio will generate a file with projects. On later stages scanio works with this list to fetch and analyze projects.  
  
Now lets fetch projects.
1. Generate ssh key and add it to your gitlab account: https://gitlab.com/-/profile/keys
2. Command example: `scanio fetch --auth-type ssh-key --ssh-key ~/.ssh/id_rsa -f /tmp/demo-group-projects.json --vcs gitlab --vcs-url gitlab.com`
As a result scanio will create folder structure `~/.scanio/projects/` and clone all projects there saving full path with namespace.
