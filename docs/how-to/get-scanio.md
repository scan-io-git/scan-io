# How to Get Scanio

To start using Scanio, there are two supported ways to install it, depending on your needs:

- Use a pre-built Docker container for an easy, isolated setup with all dependencies pre-installed. It's recommended way!
- Build the Docker container or binaries directly from the code.

## Table of Contents

- [Option 1: Docker Container](#option-1-docker-container)
  - [Prerequisites](#prerequisites)
  - [Installation Steps](#installation-steps)
- [Option 2: Manual Build](#option-2-manual-build)
  - [Build Docker Container](#build-docker-container)
  - [Build Binaries](#build-binaries)


## Option 1: Docker Container 

Scanio is distributed as a Docker container, which is the recommended method for most users. It provides a ready-to-use environment with built-in security scanner dependencies.

> [!TIP]  
> You can view the current list of bundled dependencies [here](#). ← TODO: add link to the dependency list.

### Prerequisites

Ensure you have the following installed:
- [Docker](https://www.docker.com/) — required to run the Scanio container. Refer to the [official installation guide](https://docs.docker.com/engine/install/).

### Installation Steps

> [!NOTE]  
> The Scanio Docker image is built for the `linux/amd64` architecture, compatible with most systems.

1. Pull the latest Scanio Docker image:
```bash
docker pull ghcr.io/scan-io-git/scan-io:latest
```

2. Run the CLI:
```bash
docker run --rm ghcr.io/scan-io-git/scan-io:latest --help
```

**Sample output:**
```
Scanio is an orchestrator that consolidates various security scanning capabilities, including static code analysis, secret detection, dependency analysis, etc.

  Learn more at: https://github.com/scan-io-git/scan-io

Usage:
  scanio [command]

Available Commands:
  analyse         Provides a top-level interface with orchestration for running a specified scanner
  fetch           Fetches repository code using the specified VCS plugin with consistency support
  help            Help about any command
  ...
```

Scanio is now ready to use!

## Option 2: Manual Build

### Build Docker Container

If you need a controlled build, you can build the Docker container directly from the source code.

Start from the repo cloning:
```bash
git clone https://github.com/scan-io-git/scan-io
cd scan-io
```

Continue with the [How to Build Scanio - Build Docker Image](build-scanio.md#option-1-build-docker-image) guide. 

### Build Binaries

> [!WARNING]
> We don't recommend you use `go install` because it demands significant custom changes to make the tool work as intended. 

If you prefer using the CLI directly instead of the Docker container, you can build the Scanio core and plugin binaries from source.

> [!NOTE]  
> Unlike the Docker version, the Go binary does not include dependencies for plugins. You must install these dependencies manually as described in the [plugin documentation](../reference/README.md#plugins).

Start from the repo cloning:
```bash
git clone https://github.com/scan-io-git/scan-io
cd scan-io
```

Continue with the [How to Build Scanio - uild Scanio Core and Plugins Binaries](build-scanio.md#option-2-build-scanio-core-and-plugins-binaries) guide. 

