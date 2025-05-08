# How to Update Scanio

You can update Scanio using one of two supported methods, depending on how you initially installed it:

- Using the pre-built Docker container (recommended for ease and consistency)
- Updating from the source code via `Makefile`

## Table of Contents

- [Prerequisites](#prerequisites)
- [Option 1: Docker Container](#option-1-docker-container)
  - [Clean Up Old Containers and Images](#clean-up-old-containers-and-images)
  - [Remove Dangling Images](#remove-dangling-images)
- [Option 2: Source Code](#option-2-source-code)
  - [Docker Container](#docker-container)
  - [Scanio Core and Plugins Binaries](#scanio-core-and-plugins-binaries)

## Prerequisites

Before proceeding, ensure that Scanio is already installed on your system. See [How to Get Scanio](get-scanio.md) for installation instructions.

## Option 1: Docker Container

1. To update Scanio when using Docker, simply pull the latest image:
```bash
docker pull ghcr.io/scan-io-git/scan-io:latest
```

2. After installation, you can confirm the update:
```bash
docker run --rm ghcr.io/scan-io-git/scan-io:latest version
```

**Sample output:**
```bash
Core Version: v1.0.0
Plugin Versions:
  github: v0.1.0 (Type: vcs)
  gitlab: v0.1.0 (Type: vcs)
  semgrep: v0.0.1 (Type: scanner)
  trufflehog: v0.0.2 (Type: scanner)
  trufflehog3: v0.0.2 (Type: scanner)
  bandit: v0.0.1 (Type: scanner)
  bitbucket: v0.1.2 (Type: vcs)
  codeql: v0.0.1 (Type: scanner)
Go Version: go1.23.4
Build Time: 2025-01-01T12:00:00Z
```

### Clean Up Old Containers and Images

If you prefer to work with a custom tag for easier command-line usage, you can retag the pulled image:
```bash
docker tag ghcr.io/scan-io-git/scan-io:latest scanio:latest
```

If you use retagging over time, outdated containers and images can accumulate. Use the following steps to clean up old Scanio resources.

> [!WARNING]  
> These commands will **stop and remove all containers and images** that include the word `scanio` in their name. Proceed with caution.

1. Stop all running containers containing "scanio":
```bash
docker ps | grep "scanio" | awk '{print $1}' | xargs docker stop
```

2. Remove all stopped containers containing "scanio":
```bash
docker ps -a | grep "scanio" | awk '{print $1}' | xargs docker rm
```

3. Remove all images tagged with "scanio":
```bash
docker images | grep "scanio" | awk '{print $3}' | xargs docker rmi -f
```

#### Remove Dangling Images
When retagging, Docker may leave behind dangling images—those without a tag. To remove them:

> [!WARNING]  
> This will remove **all** dangling images on your system, not just those related to Scanio.

```bash
docker image prune
```

## Option 2: Source Code

### Docker Container 

You might buld the Docker container from the source code. Start from the repo cloning:
```bash
git clone https://github.com/scan-io-git/scan-io
cd scan-io

# Sample output:
Cloning into 'scan-io'...
```

Remove local Scanio Docker images by using `Makefile`:
```bash
make clean-docker-images

# Sample output:
make clean-docker-images
Removing Docker images...
docker rmi -f scanio:1.0.0 scanio:latest || true
Untagged: scanio:latest
Deleted: sha256:5d83088bb9267431f5208d5b6fac845c126fc98923ec5e72737991fe06315760
```

Build new Docker image:
```bash
make docker

# Sample output:
Build local Docker image (no registry push)...
docker build -t scanio .
[+] Building 34.8s (31/31) FINISHED   
 => [internal] ... 
```

This command will build a local Docker image called `scanio:latest`.

Check the update results: 

```bash
docker run --rm scanio version

# Sample output:
Core Version: v0.1.0
Plugin Versions:
  github: v0.1.0 (Type: vcs)
  gitlab: v0.1.0 (Type: vcs)
  semgrep: v0.0.1 (Type: scanner)
  trufflehog: v0.0.2 (Type: scanner)
  trufflehog3: v0.0.2 (Type: scanner)
  bandit: v0.0.1 (Type: scanner)
  bitbucket: v0.1.2 (Type: vcs)
  codeql: v0.0.1 (Type: scanner)
Go Version: go1.23.4
Build Time: 2025-01-01T12:00:00Z
```


### Scanio Core and Plugins Binaries

> [!NOTE]  
> Unlike the Docker version, the Go binary does not update the dependencies for plugins automatically. You must update these dependencies manually as described in the [plugin documentation](../reference/README.md#plugins).


If you prefer using the CLI directly instead of the Docker container, you can build the Scanio core and plugin binaries from source.
```bash
git clone https://github.com/scan-io-git/scan-io
cd scan-io

# Sample output:
Cloning into 'scan-io'...
```

To build a core CLI and plugin at once, use:
```bash
make build

# Sample output:
make build
Building Scanio core...
go build -ldflags="-X 'github.com/scan-io-git/scan-io/cmd/version.CoreVersion=1.0.0' \
                           -X 'github.com/scan-io-git/scan-io/cmd/version.GolangVersion=go1.23.4' \
                           -X 'github.com/scan-io-git/scan-io/cmd/version.BuildTime=2025-01-01T12:00:00Z'" \
           -o ~/.local/bin/scanio . || exit 1
Scanio core built successfully!
Building Scanio plugins...
Building plugin 'bandit' (v0.0.1, type: scanner)
...
All Scanio plugins built successfully!
```

Verify installation:
```bash
scanio --version

# Sample output:
scanio version
Core Version: v1.0.0
Plugin Versions:
...
  bandit: v0.0.1 (Type: scanner)
...
Go Version: go1.23.4
Build Time: 2025-01-01T12:00:00Z
```

In this scenario the CLI core binary is intalled into `~/.local/bin/scanio`, plugins are into `~/.scanio/plugins/`.