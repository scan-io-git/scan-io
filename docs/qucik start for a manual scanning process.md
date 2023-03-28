# Quick Start for a Manual scanning process for AppSec teams and developers
This scenario involves performing on-demand scanning, which is a common use case for the application. It allows you to manually control any arguments for the scanner that you need, including implementation review scanning, developing custom rules and scanning, scans initiated by developers to self-check, and more.<br><br>

To accomplish this, you have two options:
- Use the command-line interface (CLI) as is.
- Use the Docker container with the CLI.<br><br>

You can run the manual scan process in different environments, such as:
- Kubernetes (K8s) jobs.
- Personal devices.<br><br>

By using this approach, AppSec teams and developers can easily perform manual scans to search for vulnerabilities, secrets, and vulnerable 3rd party dependencies, and get the information they need to improve the security of their applications.

## Working on a Personal Device
You can use Docker with the application or CLI as is. Here are the commands for both methods.
> We're using Bitbucket and Semgrep as examples, but you can easily change the VCS and scanner flags to suit your needs. To see a list of supported plugins for your command, check out the [README](../README.md#articles-for-read).

<br>

### Prerequisites
The first step is to build/pull/install the container/CLI. Check the [installation guide](../README.md#installation).<br>

Depending on your VCS and scanner, you may need different prerequisites. Check out the requirements for your plugin [here](../README.md#articles-for-read).<br><br>

In the case of Bitbucket, you will need to add a few environment variables
```
# SCANIO_BITBUCKET_USERNAME - will be used as your username for the API
export SCANIO_BITBUCKET_USERNAME=<username>

# SCANIO_BITBUCKET_TOKEN - will be used to access the API
export SCANIO_BITBUCKET_TOKEN=<token>
```
<br>

If you use an SSH key for authentication, you need to use a password for your private key.<br>
> We recommend using ssh-agent or other types of SSH keys for fetching!
```
# SCANIO_BITBUCKET_SSH_KEY_PASSWORD - will be used to access your protected private key
SCANIO_BITBUCKET_SSH_KEY_PASSWORD=<password>
```
<br>

For working with a Docker container, we will use the ```-e``` flag to copy variables to the container environment.
> You also can use a ```--env-file ./env.list```.
```
-e SCANIO_BITBUCKET_USERNAME='<username>' \
-e SCANIO_BITBUCKET_TOKEN \
-e SCANIO_BITBUCKET_SSH_KEY_PASSWORD .....
```
<br>

We also need to mount a file system to the container. ```/data``` is the default directory for Scanio home in Docker. We will use the ```-v``` flag for this purpose:
```
-v "/~/development/:/data"
```
<br>

If you use an SSH key for authentication, you need to copy your private key to the container:
```
-v "/~/.ssh/id_ed25519:/data/id_ed25519"
```
<br>

If you use an SSH agent for authentication, you need to copy the socket to the container:
```
ssh-add private.key 

# For Linux Docker's flags
-e SSH_AUTH_SOCK='/ssh-agent' \
-v "$SSH_AUTH_SOCK:/ssh-agent"  

# For Mac Docker's flags 
-e SSH_AUTH_SOCK="/run/host-services/ssh-auth.sock" \
-v /run/host-services/ssh-auth.sock:/run/host-services/ssh-auth.sock
```
[MacIssue](https://github.com/docker/for-mac/issues/410#issuecomment-577064671) with mouting an SSH socket.

<br>

### Fetching a particular repository
To fetch a particular repository for scanning, you can use the following commands:<br><br>

Command for CLI.
```
scanio fetch --vcs bitbucket --auth-type ssh-agent -j 1 https://git.acronis.com/projects/SEC/repos/passport/browse
```
<br>

For Docker.
```
docker run --rm -e SCANIO_BITBUCKET_USERNAME='john.doe' \
-e SCANIO_BITBUCKET_TOKEN \
-e SCANIO_BITBUCKET_SSH_KEY_PASSWORD \
-v "/~/development/:/data" \
-v "/~/.ssh/id_ed25519:/data/id_ed25519" \
scanio fetch --vcs bitbucket --auth-type ssh-agent -j 1 https://example.com/projects/SCANIO/repos/scanio/browse
```

> Replace ```john.doe``` with your Bitbucket username, ```https://example.com/projects/SCANIO/repos/scanio/browse``` with the URL of the repository you want to fetch and `/~/development/:/data` with path of the repository to scan. If you use SSH key authentication, make sure to copy your private key to the container or use an SSH agent authentification.

<br>

### Analyzing a particular repository
To analyze a particular repository, use the following commands:<br><br>

Command for CLI.
```
scanio analyse --scanner semgrep /data/projects/example.com/scanio/scanio
```
<br>

Command for docker:
```
docker run --rm \
-v "/~/development/:/data" \
scanio analyse --scanner semgrep /data/projects/example.com/scanio/scanio
```

The analysis result can be found in the same folder on your host file system: ```/~/development/projects/example.com/scanio/scanio/semgrep```.

> Replace ```/data/projects/example.com/scanio/scanio``` with the path of the repository you want to scan.

### Interactive mode with bash for Docker
To work with the Scanio container in interactive mode, you can use the following command:
```
docker run --rm \
-e SCANIO_BITBUCKET_USERNAME='john.doe' \
-e SCANIO_BITBUCKET_TOKEN \
-e SCANIO_BITBUCKET_SSH_KEY_PASSWORD \
-v "/~/development/:/data" \
-v "/~/.ssh/id_ed25519:/data/id_ed25519" \
--entrypoint /bin/bash -it scanio  
```
<br>

This command sets the necessary environment variables and mounts the appropriate directories for working with Scanio. Additionally, it starts an interactive bash shell inside the container, which allows you to run commands and interact with the container's file system.

> Replace ```john.doe``` with your Bitbucket username and `/~/development/:/data` with path of the repository to scan. If you use SSH key authentication, make sure to copy your private key to the container or use an SSH agent authentification.

## Working on a Remote k8s Cluster
*In progress...*
