# Makefile

This page describes the available targets and variables in the Scanio [`Makefile`](../../Makefile). 

The [`Makefile`](../../Makefile) automates building, cleaning, and managing artifacts locally for:
- Scanio CLI core binary
- Plugin binaries
- Docker image
- Python environment for rule folder compile

## Table of Contents

- [Variables](#variables)
- [Targets and Descriptions](#targets-and-descriptions)
- [Requirements](#requirements)
- [Actions](#actions)
    - [Call Help](#call-help)
    - [Build Scanio CLI Core and Plugins](#build-scanio-cli-core-and-plugins)
        - [Build Scanio CLI Core](#build-scanio-cli-core)
        - [Build Scanio Plugins](#build-scanio-plugins)
    - [Create Plugin Directory](#create-plugin-directory)
    - [Build Scanio Docker Container](#build-scanio-docker-container)
        - [Build Scanio Docker Container with Custom Variables](#build-scanio-docker-container-with-custom-variables)
    - [Push Docker Container](#push-docker-container)
    - [Setup Python Env for Rules Compile](#setup-python-env-for-rules-compile)
    - [Rules Compile](#rules-compile)
    - [Clean Build Artifacts](#clean-build-artifacts)
    - [Clean Docker Images](#clean-docker-images)
    - [Clean Python Venv](#clean-python-venv)
    - [Run Go Tests](#run-go-tests)


## Variables

Variables can be overridden by setting them via the command line.

| Variable          | Default                                           | Purpose                                                          |
|-------------------|---------------------------------------------------|------------------------------------------------------------------|
| `CORE_BINARY`      | `~/.local/bin/scanio`                            | Path to output Scanio CLI binary                              |
| `PLUGINS_DIR`      | `~/.scanio/plugins/`                             | Directory where built plugins are stored                      |
| `REGISTRY`         | *(empty)*                                        | Docker registry URL (used in `docker-build`, `docker-push`)   |
| `IMAGE_NAME`       | `scanio`                                         | Name of the Docker image                                      |
| `IMAGE_TAG`        | `scanio`, or `<registry>/scanio` if REGISTRY set | Full Docker image tag                                         |
| `TARGET_OS`        | `linux`                                          | Target OS for Docker builds                                   |
| `TARGET_ARCH`      | `amd64`                                          | Target architecture for Docker builds                         |
| `PLATFORM`         | `linux/amd64`                                    | Platform specifier for Docker builds                          |
| `PLUGINS`          | `github gitlab bitbucket semgrep bandit trufflehog` | List of plugins for build                                  |
| `VERSION`          | Extracted from `VERSION` file (via `jq`)         | Core version information used in build metadata               |
| `GO_VERSION`       | Output of `go version` command                   | Go version for embedding in build metadata                    |
| `BUILD_TIME`       | Current UTC time (RFC3339 format)                | Timestamp embedded in builds                                  |
| `RULES_SCRIPT`     | `scripts/rules/rules.py`                         | Python script to generate security rule sets                  |
| `RULES_CONFIG`     | `scripts/rules/scanio_rules.yaml`                | Configuration for rules generating                            |
| `RULES_DIR`        | `./rules`                                        | Directory containing for generated security rule sets         |
| `VENV_DIR`         | `.venv`                                          | Path to Python virtual environment folder                     |
| `REQUIREMENTS_FILE`| `scripts/rules/requirements.txt`                 |Requirements file used to install rule generation dependencies |
| `USE_VENV`         | `false`                                          | Whether to use a virtual environment for Python rule generation |

## Targets and Descriptions

| Target                | Description |
|------------------------|------------------------------------------------------------|
| `build`                | Build Scanio CLI core and plugins                          |
| `build-cli`            | Build the Scanio CLI core binary                           |
| `build-plugins`        | Build Scanio plugins                                       |
| `build-rules`          | Build  custom rule sets using Python script                |
| `clean`                | Clean all artifacts (plugins, Docker images, Python env)   |
| `clean-plugins`        | Clean plugin directory                                     |
| `clean-docker-images`  | Clean local Scanio Docker images                           |
| `docker`               | Build local Docker image (no registry push)                |
| `docker-build`         | Build Docker image (tagged by version and latest)          |
| `docker-push`          | Push Docker image to registry                              |
| `help`                 | Display available commands                                 |    
| `prepare-plugins   `   | Prepare plugin directory                                   |              
| `setup-python-env`     | Set up Python virtual environment and install dependencies |
| `test`                 | Run Go tests                                               |

## Requirements

The Makefile includes automatic dependency validation:
- **[Go](https://go.dev/doc/install)**: Required to build CLI and plugins
- **[jq](https://stedolan.github.io/jq/download/)**: Required for plugin metadata parsing (`VERSION` files)
- **[Docker](https://docs.docker.com/get-docker/)**: Required for building Docker images
- **[Python3 + pip](https://www.python.org/downloads/)**: Required if building custom rule sets

If any dependency is missing, a helpful error message will be printed and the build will halt.

## Actions
### Call Help

Display available commands and descriptions.

```bash
make help
```

**Sample output:**
```bash
build-cli                      Build Scanio CLI core binary
build-plugins                  Build Scanio plugins
build-rules                    Build custom rule sets using Python script
...
```

### Build Scanio CLI Core and Plugins

> [!NOTE]  
> Unlike the Docker version, the Go binary does not include dependencies for plugins. You must install these dependencies manually as described in the [plugin documentation](../reference/README.md#plugins).

Build both the Scanio CLI core binary and plugins.

```bash
make build
```

**Variables supported**
- `VERSION`
- `GO_VERSION`
- `BUILD_TIME`
- `CORE_BINARY`
- `PLUGINS_DIR`
- `PLUGINS`

**Sample output:**
```bash
Building Scanio CLI core...
go build -ldflags="-X 'github.com/scan-io-git/scan-io/cmd/version.CoreVersion=1.0.0' \
                           -X 'github.com/scan-io-git/scan-io/cmd/version.GolangVersion=go1.23.4' \
                           -X 'github.com/scan-io-git/scan-io/cmd/version.BuildTime=2025-01-01T12:00:00Z'" \
           -o ~/.local/bin/scanio . || exit 1
Scanio CLI core built successfully!
Building Scanio plugins...
Building plugin 'bandit' (v0.0.1, type: scanner)
...
All Scanio plugins built successfully!
```

#### Build Scanio CLI Core

The action builds the CLI core only:

```bash
make build-cli
```

**Variables supported:**
- `VERSION`
- `GO_VERSION`
- `BUILD_TIME`
- `CORE_BINARY`
- `PLUGINS`

The binary will be saved by default to:
```
~/.local/bin/scanio
```

You can customize the target location by setting the `CORE_BINARY` variable:
```bash
make build-cli CORE_BINARY=/path/to/scanio
```

**Sample output:**
```bash
Building Scanio CLI core...
go build -ldflags="-X 'github.com/scan-io-git/scan-io/cmd/version.CoreVersion=1.0.0' \
                           -X 'github.com/scan-io-git/scan-io/cmd/version.GolangVersion=go1.23.4' \
                           -X 'github.com/scan-io-git/scan-io/cmd/version.BuildTime=2025-01-01T12:00:00Z'" \
           -o ~/.local/bin/scanio . || exit 1
Scanio CLI core built successfully!
```

#### Build Scanio Plugins

> [!NOTE]  
> Unlike the Docker version, the Go binary does not include dependencies for plugins. You must install these dependencies manually as described in the [plugin documentation](../reference/README.md#plugins).

Plugins are built and saved into a structured plugin directory:

```bash
make build-plugins
```

**Variables supported:**
- `PLUGINS_DIR`
- `GO_VERSION`
- `BUILD_TIME`

By default, plugins are installed into:
```
~/.scanio/plugins/
```

You can customize the plugin directory path by setting the `PLUGINS_DIR` variable:
```bash
make build-plugins PLUGINS_DIR=/path/to/plugin_folder/
```

**Sample output:**
```bash
Building Scanio plugins...
Building plugin 'bandit' (v0.0.1, type: scanner)
...
All Scanio plugins built successfully!
```

### Create Plugin Directory

Create the plugin directory structure if it doesn't exist.
```bash
make prepare-plugins
```

**Variables supported:**
- `PLUGINS_DIR`

**Sample output:**
```bash
Preparing plugin directory - ~/.scanio/plugins
```

### Build Scanio Docker Container

> [!NOTE]  
> This command builds a Docker container using default settings derived from your local environment (OS, architecture, etc.).

To build docker container:
```bash
make docker
```

This command will build a local Docker image called `scanio:latest`.

**Variables supported:**
- `IMAGE_NAME`
- `PLUGINS`

**Sample output:**
```bash
Build local Docker image (no registry push)...
docker build -t scanio .
[+] Building 34.8s (31/31) FINISHED   
 => [internal] ... 
```

#### Build Scanio Docker Container with Custom Variables

To build docker container with custom variables like the image use:
```bash
make docker-build VERSION=1.2 REGISTRY=my.registry.com/scanio
```

**Variables supported:**
- `IMAGE_NAME`
- `REGISTRY`
- `IMAGE_TAG`
- `TARGET_OS`
- `TARGET_ARCH`
- `VERSION`
- `PLUGINS`

**Sample output:**
```bash
Building Docker image for linux/amd64...
docker build --build-arg TARGET_OS=linux --build-arg TARGET_ARCH=amd64 --platform=linux/amd64 \
        -t my.registry.com/scanio/scanio:1.2 -t my.registry.com/scanio/scanio:latest . || exit 1
[+] Building 34.8s (31/31) FINISHED  
 => [internal]
```

### Push Docker Container

Push built images to a Docker registry:
```bash
make docker-push REGISTRY=my.registry.com/scanio VERSION=1.2
```

**Variables supported:**
- `IMAGE_TAG`
- `IMAGE_NAME`
- `REGISTRY`
- `VERSION`

**Sample output:**
```bash
Pushing Docker image to my.registry.com/scanio...
docker push my.registry.com/scanio/scanio:1.2 || exit 1
The push refers to repository [my.registry.com/scanio/scanio]
```

### Setup Python Env for Rules Compile

Set up a Python virtual environment and install dependencies from requirements.txt for further rules compile.

```bash
make setup-python-env
```

**Variables supported:**
- `VENV_DIR`
- `REQUIREMENTS_FILE`

**Sample output:**
```bash
Setting up Python virtual environment in .venv...
...
Python virtual environment setup complete.
```

### Rules Compile

Generate custom rule sets using the rules script.
```bash
make build-rules
```

> [!NOTE]  
> For more information Refer to the guide How to Build Rule Sets (TODO) and Reference of Custom Script builder (TODO)

Result will be save to `./rules` by default. 

**Variables supported:**
- `USE_VENV`
- `VENV_DIR`
- `RULES_SCRIPT`
- `RULES_CONFIG`
- `RULES_DIR`
- `FORCE` - forcefully clean the `rules` directory without confirmation 
- `VERBOSE` - verbose output of the python script

**Sample output:**
```bash
todo
```


### Clean Build Artifacts

To remove generated CLI binary, plugins, Docker images, and Python virtual environments:

```bash
make clean
```

**Sample output:**
```bash
Cleaning CLI core binary - ~/.local/bin/scanio
Cleaning plugin directory - ~/.scanio/plugins
Removing Docker images...
docker rmi -f scanio:1.0.0 scanio:latest || true
...
Cleaning Python virtual environment...
rm -rf .venv
```

### Clean CLI Core

Remove CLI core binary:

```bash
make clean-cli
```

**Variables supported:**
- `CORE_BINARY`

**Sample output:**
```bash
Cleaning CLI core binary - ~/.local/bin/scanio
Cleaning plugin directory - ~/.scanio/plugins
Removing Docker images...
docker rmi -f scanio:1.0.0 scanio:latest || true
...
Cleaning Python virtual environment...
rm -rf .venv
```

### Clean Plugins 

Remove all plugin binaries:

```bash
make clean-plugins
```

**Variables supported:**
- `PLUGINS_DIR`

**Sample output:**
```bash
Cleaning plugin directory - ~/.scanio/plugins
```

### Clean Docker Images

Remove local Scanio Docker images:
```bash
make clean-docker-images
```

**Variables supported:**
- `IMAGE_NAME`
- `VERSION`

**Sample output:**
```bash
Removing Docker images...
docker rmi -f scanio:1.0.0 scanio:latest || true
Untagged: scanio:latest
Deleted: sha256:5d83088bb9267431f5208d5b6fac845c126fc98923ec5e72737991fe06315760
```

### Clean Python Venv

Remove the Python virtual environment.

```bash
make clean-python-env
```

**Variables supported:**
- `VENV_DIR`

**Sample output:**
```bash
Cleaning Python virtual environment...
rm -rf .venv
```

### Run Go Tests

Run Go tests across the Scanio project.
```bash
make test
```

**Sample output:**
```bash
go test -v ./... && echo "All tests passed"
go: downloading github.com/stretchr/testify v1.10.0
go: downloading github.com/pmezard/go-difflib v1.0.0
...
PASS
ok      github.com/scan-io-git/scan-io/pkg/shared/vcsurl        1.213s
All tests passed
```

