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

## Step 2. K8S
The main difference is that we have to deliver our secrets (token and ssh key) into pod inside the k8s.  
We use helm chart to deploy this type of resources. Helm charts allow us to be flexibale, so everybody can deliver secrets in a convinient way: for example via k8s secrets, or vault sidecar, or any other mechanism.  
Let's look at the example with k8s secrets:
```bash
❯ kubectl create secret generic gitlab-token --from-literal=token=[...redacted...]

❯ kubectl create secret generic ssh-key --from-file=private=/path/to/private_key --from-file=public=/path/to/public_key
```
Change help chart, for example, `scanio-main-pod` by adding the following lines, which will set env var from secret:
```yaml
- name: GITLAB_TOKEN
  valueFrom:
    secretKeyRef:
    name: gitlab-token
    key: token
```
add the following lines to mount secret with ssh key as a volume:
```yaml
volumes:
- name: ssh-key-volume
  secret:
    secretName: ssh-key

[...redacted...]

containers:
  [...redacted...]
  volumeMounts:
  - name: ssh-key-volume
    readOnly: true
    mountPath: "/ssh-key-volume"
```

Before the next step, don't forget to build docker image and push to a registry, that k8s works with.

Deploy the pod:  
`helm install scanio-main-pod helm/scanio-main-pod/`

Get shell inside the pod:  
`kubectl exec -it scanio-main-pod -- bash`

Now interact with scanio:  
```bash
scanio list --vcs gitlab --vcs-url gitlab.com --namespace demo-group --output /tmp/demo-group-projects.json

scanio fetch --vcs gitlab --vcs-url gitlab.com --auth-type ssh-key --ssh-key /ssh-key-volume/private -f /tmp/demo-group-projects.json
```
As a result you will find fetched project in `/data/projects` folder.
