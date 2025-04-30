# How to Build Scanio

This guide explains how to build the Scanio CLI, its plugins, and optional Docker image using the provided `Makefile`. There are two supported ways to build it, depending on your needs:

- Build a Docker container for an easy, isolated setup with all dependencies pre-installed. It's recommended way!
- Build a Go binary from the source code.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Option 1: Build Docker Image](#option-1-build-docker-image)
- [Option 2: Build Scanio Core and Plugins Binaries](#option-2-build-scanio-core-and-plugins-binaries)
    - [Build Scanio Core Separately](#build-scanio-core-separately)
    - [Build Scanio Plugins Separately](#build-scanio-plugins-separately)

## Prerequisites

Ensure the following dependencies are installed on your system before building Scanio:

- [Go Programming Language](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/get-docker/) (optional, for Docker image build)
- [jq](https://stedolan.github.io/jq/download/) (required for plugin builds)

Scanio’s Makefile automatically checks for missing dependencies and provides an error message if anything is missing.

You should clone the code from the repo as the first step for any options:
```bash
git clone https://github.com/scan-io-git/scan-io
cd scan-io
```

## Option 1: Build Docker Image

If you want a fully containerized version of Scanio with all dependencies bundled:

```bash
make docker

# Sample output:
Build local Docker image (no registry push)...
docker build -t scanio .
[+] Building 34.8s (31/31) FINISHED   
 => [internal] ... 
```

This command will build a local Docker image called `scanio:latest`.

> [!TIP]
> You can customize the image name by setting the `IMAGE_NAME` and `REGISTRY` variables.

If you need to specify a registry and a version:
```bash
make docker-build VERSION=1.2 REGISTRY=my.registry.com/scanio

# Sample output:
make docker-build VERSION=1.2 REGISTRY=my.registry.com/scanio
Building Docker image for linux/amd64...
docker build --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 --platform=linux/amd64 \
        -t my.registry.com/scanio/scanio:1.2 -t my.registry.com/scanio/scanio:latest . || exit 1
[+] Building 34.8s (31/31) FINISHED  
 => [internal]
```

Push built images to a Docker registry:
```bash
make docker-push REGISTRY=my.registry.com/scanio VERSION=1.2

# Sample output:
Pushing Docker image to my.registry.com/scanio...
docker push my.registry.com/scanio/scanio:1.2 || exit 1
The push refers to repository [my.registry.com/scanio/scanio]
```

## Option 2: Build Scanio Core and Plugins Binaries

> [!NOTE]  
> Unlike the Docker version, the Go binary does not include dependencies for plugins. You must install these dependencies manually as described in the [plugin documentation](../reference/README.md#plugins).

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

### Build Scanio Core CLI Separately 
If you need to build the core CLI only, use:

```bash
make build-cli

# Sample output:
Building Scanio core...
go build -ldflags="-X 'github.com/scan-io-git/scan-io/cmd/version.CoreVersion=1.0.0' \
                           -X 'github.com/scan-io-git/scan-io/cmd/version.GolangVersion=go1.23.4' \
                           -X 'github.com/scan-io-git/scan-io/cmd/version.BuildTime=2025-01-01T12:00:00Z'" \
           -o ~/.local/bin/scanio . || exit 1
Scanio core built successfully!
```

The binary will be saved by default to:
```
~/.local/bin/scanio
```

You can customize the target location by setting the `CORE_BINARY` variable:
```bash
make build-cli CORE_BINARY=/path/to//scanio
```

## Build Scanio Plugins Separately

> [!NOTE]  
> Unlike the Docker version, the Go binary does not include dependencies for plugins. You must install these dependencies manually as described in the [plugin documentation](../reference/README.md#plugins).

Plugins are built separately and saved into a structured plugin directory.

Build plugins only:

```bash
make build-plugins

# Sample output:
Building Scanio plugins...
Building plugin 'bandit' (v0.0.1, type: scanner)
...
All Scanio plugins built successfully!
```

By default, plugins are installed into:
```
~/.scanio/plugins/
```

You can customize the plugin directory path using by setting the `PLUGINS_DIR` variable::
```bash
make build-plugins PLUGINS_DIR=/path/to/plugin_folder/
```


