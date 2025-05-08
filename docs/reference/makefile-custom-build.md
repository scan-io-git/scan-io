# Makefile for Custom Build

This page describes the available targets and variables in the [`Makefile`](../../scripts/custom-build/Makefile). This [`Makefile`](../../scripts/custom-build/Makefile) supports custom deployments of Scanio, including cases where users have their own versions of Scanio, plugins, and custom rule sets. 

This workflow automates:
- Cloning the Scanio source repository: by the repo you can customized any aspect of the tool, for example, have your own set of custom plugins.
- Compiling and copying custom rule sets.
- Building and pushing a Docker image.

## Table of Contents

- [Variables](#variables)
- [Targets and Descriptions](#targets-and-descriptions)
- [Requirements](#requirements)
- [Custom Build Repository Structure](#custom-build-repository-structure)
- [Custom Build Workflow](#custom-build-workflow)
- [Actions](#actions)
    - [Call Help](#call-help)
    - [Clone Scanio Repository](#clone-scanio-repository)
    - [Copy Config File](#copy-config-file)
    - [Copy Rule Set](#copy-rule-set)
    - [Build Rule Set](#build-rule-set)
    - [Build Docker Image](#build-docker-image)
    - [Push Docker Image](#push-docker-image)
    - [Clean Local Artifacts](#clean-local-artifacts)
    - [Full Build Process](#full-build-process)

## Variables

Variables can be overridden by setting them via the command line.

| Variable          | Default                                           | Purpose                                                       |
|-------------------|---------------------------------------------------|---------------------------------------------------------------|
| `SCANIO_REPO`      | `https://github.com/scan-io-git/scan-io.git`     | URL of the Scanio Git repository                              |
| `SCANIO_REPO_DIR`  | `./scan-io`                                      | Directory for cloning the Scanio repository                   |
| `RULES_CONFIG`     | `scanio_rules.yaml`                              | Path to the custom rule set config                            |
| `SCANIO_CONFIG`     | `config.yaml`                                   | Path to the custom Scanio core config                         |
| `CLONED_CONFIG_PATH`| `$(SCANIO_REPO_DIR)/`                           | Destination path for Scanio core config file in the cloned repo    |
| `CLONED_RULES_PATH`| `$(SCANIO_REPO_DIR)/scripts/rules/$(RULES_CONFIG)` | Destination path for the rule set config file in the cloned repo |
| `PLUGINS`          | `github gitlab bitbucket semgrep bandit trufflehog` | List of plugins for build                                  |
| `VERSION`          | `1.0`                                            | Docker image version                                          |
| `TARGET_OS`        | `linux`                                          | Target OS for Docker builds                                   |
| `TARGET_ARCH`      | `amd64`                                          | Target architecture for Docker builds                         |
| `REGISTRY`         | (empty)                                          | Docker registry (optional)                                    |

## Targets and Descriptions

| Target                | Description |
|------------------------|------------------------------------------------------------|
| `build`                | Orchestrate full build + push process (clone, copy, build, push)  |
| `build-docker`         | Build a Docker image with embedded rules                   |
| `build-rules`          | Build rule sets in the cloned repo                         |
| `clean`                | Clean up cloned repository and generated artifacts         |
| `clone-scanio-repo`    | Clone the Scanio Git repository                            |
| `copy-config`          | Copy configuration file to cloned repo                     |
| `copy-rules`           | Copy rule set to Scanio cloned repository                  |
| `clean`                | Clean up cloned repository and generated artifacts         |
| `push-docker`          | Push the Docker image to the configured registry           |
| `help`                 | Show usage help and variables                              |


## Requirements

The `Makefile` for custom build uses a `Makefile` from the cloned repo, by default it's [`Makefile`](../../Makefile). The default build file provide dependency check. For more information refer to [Reference Makefile](makefile.md#dependency-checks).

## Custom Build Repository Structure
The files from the [scripts/custom-build](../../scripts/custom-build/) directory must be placed either in your custom Git repository or on your local machine.

Currently, the custom build setup supports the following files:
- `config.yml` — A global Scanio core configuration file. This will override the default config from the cloned Scanio repository.
- `scanio_rules.yaml` — A configuration file specifying tools and rules for automated rule set building.
- `Makefile` — A script for building a custom Docker image with your defined global config and custom rule sets.

## Custom Build Workflow
First, clone your custom build repository containing the files listed above or ensure they are present locally on your disk.


To run a fully automated build, use the `make build` command. You can pass necessary arguments at runtime (see the full list under [Variables](#variables)).

```bash
git clone https://my.internal.git/security/scanio-build
cd scanio-build
make build SCANIO_REPO=https://my.internal.github.com/security/scanio-code VERSION=1.0 REGISTRY=my.registry.com/scanio
```

### Things to Consider:
1. **Custom Plugins**: If you have custom plugins, they can be added to the `plugins/` directory in the cloned repository and will be built as part of the Scanio plugins. Also, you can use the `PLUGINS` argument include particular plugins into your docker image.
2. **Customization**: You can modify the paths, Docker image versions, and other settings by overriding the default values with command-line variables.

### Step-by-Step Breakdown
-> `clone-scanio-repo`

This step clones the Scanio source code repository. You may specify a custom repository with the `SCANIO_REPO` argument, or omit it to use the [official repository](https://github.com/scan-io-git/scan-io).

The code is cloned into the `./scan-io` directory by default, unless overridden with the `SCANIO_REPO_DIR` argument.

-> `copy-config`

Next, the `config.yml` file in the cloned repository is replaced with the one from your local directory. 

Default source: `config.yml` (can be overridden with `SCANIO_CONFIG`)

Default destination: `./scan-io/config.yml` (can be overridden with `CLONED_CONFIG_PATH`)

-> `copy-rules`

The rule set configuration file (`scanio_rules.yaml`) is copied into the rules/ directory of the cloned repository.

Default source: `scanio_rules.yaml` (can be overridden with `RULES_CONFIG`)

Default destination: `./scan-io/scripts/rules/scanio_rules.yaml` (can be overridden with `CLONED_RULES_PATH`)

-> `build-rules`

This step compiles rule sets using the copied `scanio_rules.yaml` file. The [rules.py](../../scripts/rules/README.md) script reads the config, clones the specified rules from repositories, and places them into the `rules/` directory inside the cloned repository.

All dependencies are installed in a temporary Python virtual environment. If the `rules/` directory already exists, it will be cleared beforehand.

-> `build-docker`

This step builds the Docker image for Scanio.

You can specify:
- `PLUGINS` (default: `github gitlab bitbucket semgrep bandit trufflehog`) - list of plugins to build
- `VERSION` (default: `1.0`)
- `TARGET_OS` (default: `linux`)
- `TARGET_ARCH` ( default: `amd64`)

-> `push-docker`

Finally, the built Docker image is pushed to a registry. You must specify the `REGISTRY` argument for this step to succeed.


## Actions
### Call Help

Display all available targets and how to configure variables.

```bash
make help
```

**Sample output:**
```bash
Usage: make <target> [options]
Options:
  SCANIO_REPO      - URL of the Scanio repo (default: https://github.com/scan-io-git/scan-io.git)
  RULES_CONFIG     - Path to the custom rule set (default: ./scanio_rules.yaml)
...
```

### Clone Scanio Repository

Clone the Scanio repository. Removes the directory first if it already exists.

```bash
make clone-scanio-repo
```

**Variables supported**
- `SCANIO_REPO_DIR`
- `SCANIO_REPO`

**Sample output:**
```bash
[Custom Makefile] Removing existing Scanio repository './scan-io' ...
[Custom Makefile] Cloning Scanio repository to './scan-io'
Cloning into './scan-io'...
remote: Enumerating objects: 4241, done.
remote: Counting objects: 100% (520/520), done.
remote: Compressing objects: 100% (215/215), done.
remote: Total 4241 (delta 356), reused 319 (delta 304), pack-reused 3721 (from 1)
Receiving objects: 100% (4241/4241), 14.75 MiB | 13.33 MiB/s, done.
Resolving deltas: 100% (2589/2589), done.
```

### Copy Config File

Copy your local Scanio config to the expected location in the cloned repo. 
```bash
make copy-config
```

**Variables supported**
- `SCANIO_CONFIG`
- `CLONED_CONFIG_PATH`

**Sample output:**
```bash
[Custom Makefile] Copying config file from config.yml to ./scan-io/...
```

### Copy Rule Set

Copies a custom rule sets (by default the rule sets are specified by the `scanio_rule.yaml` in the same directort) into the Scanio rules folder inside the cloned repo.

```bash
make copy-rules
```


**Variables supported**
- `RULES_CONFIG`
- `CLONED_RULES_PATH`

**Sample output:**
```bash
[Custom Makefile] Copying custom rule set from scanio_rules.yaml to ./scan-io/scripts/rules/scanio_rules.yaml...
```

### Build Rule Set

Runs Scanio's make build-rules internally in the cloned repository using a Python virtual environment.

```bash
make build-rules
```

**Variables supported**
- `SCANIO_REPO_DIR`

**Sample output:**
```bash
[Custom Makefile] Building rule sets in ./scan-io...
Setting up Python virtual environment in .venv...
Collecting pip
...
Force cleaning the rules directory '/tmp/scan-io/scripts/custom-build/scan-io/rules' (without confirmation).
Cleaned up rules directory '/tmp/scan-io/scripts/custom-build/scan-io/rules'.
Using temporary directory: /var/folders/4s/w06n74097mjct38rcxfpf3zm0000gn/T/tmpbl5bvhj4
Processing tools:   0%|                                                                                                                                            | 0/1 [00:00<?, ?tool/s]Processing tool: semgrep
  Processing ruleset: default
    Processing rules from: https://github.com/semgrep/semgrep-rules.git
    Cloning https://github.com/semgrep/semgrep-rules.git (branch: develop) into /var/folders/4s/w06n74097mjct38rcxfpf3zm0000gn/T/tmpbl5bvhj4/semgrep/8ac2e8ca-0488-4ecd-9313-c6c3495a4785
      Processing: 100%|██████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████| 1/1 [00:00<00:00, 1021.26file/s]
    Processing rules from: https://github.com/trailofbits/semgrep-rules.git                                                                                        | 0/1 [00:00<?, ?file/s]
    Cloning https://github.com/trailofbits/semgrep-rules.git (branch: main) into /var/folders/4s/w06n74097mjct38rcxfpf3zm0000gn/T/tmpbl5bvhj4/semgrep/bf64cfd9-e4d9-429e-a886-563d766e6c32
      Processing: 100%|███████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████| 1/1 [00:00<00:00, 969.11file/s]
    Backup of tool-specific YAML saved to ./rules/semgrep/default/scanio_rules.yaml.back                                                                           | 0/1 [00:00<?, ?file/s]
    Finished processing ruleset: default
Finished processing tool: semgrep
Processing tools: 100%|████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████| 1/1 [00:04<00:00,  4.49s/tool]
Backup of the entire scanio_rules.yaml saved to ./rules/scanio_rules.yaml

Temporary directory cleaned up automatically.
Custom rule sets built successfully!
```

### Build Docker Image

Builds the Docker image from the cloned repo.

```bash
make build-docker 
```

**Variables supported**
- `VERSION`
- `PLUGINS`
- `TARGET_OS`
- `TARGET_ARCH`
- `REGISTRY`

**Sample output:**
```bash
[Custom Makefile] Building Docker image for linux/amd64...
Building Docker image for linux/amd64...
docker build --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 --platform=linux/amd64 \
	-t scanio:1.0 -t scanio:latest . || exit 1
[+] Building 34.8s (31/31) FINISHED   
 => [internal] ... 
```

### Push Docker Image

Push the built image to the configured Docker registry.

```bash
make push-docker REGISTRY=example.com/scanio
```

**Variables supported**
- `SCANIO_REPO_DIR`
- `VERSION`
- `REGISTRY`

**Sample output:**
```bash
[Custom Makefile] Pushing Docker image to:  my.registry.com/scanio...
Pushing Docker image to: my.registry.com/scanio...
docker push my.registry.com/scanio/scanio:1.2 || exit 1
The push refers to repository [my.registry.com/scanio/scanio]
```

### Clean Local Artifacts

Remove the entire cloned repository and any intermediate artifacts.

```bash
make clean
```

**Variables supported**
- `SCANIO_REPO_DIR`

**Sample output:**
```bash
[Custom Makefile] Cleaning up './scan-io'...
```

### Full Build Process

This is the main orchestration entry point if you want to run the full pipeline.
Clones the repo, copies config and rules, builds rule sets, builds and pushes the Docker image.
```bash
make build REGISTRY=example.com/scanio
```

**Variables supported**
- `SCANIO_REPO`
- `SCANIO_REPO_DIR`
- `SCANIO_CONFIG`
- `RULES_CONFIG`
- `CLONED_CONFIG_PATH`
- `CLONED_RULES_PATH`
- `VERSION`
- `TARGET_OS`
- `TARGET_ARCH`
- `REGISTRY`

**Sample output:**
```bash
[Custom Makefile] Cloning Scanio repository to './scan-io'
Cloning into './scan-io'...
remote: Enumerating objects: 4241, done.
remote: Counting objects: 100% (520/520), done.
remote: Compressing objects: 100% (216/216), done.
remote: Total 4241 (delta 356), reused 318 (delta 303), pack-reused 3721 (from 1)
Receiving objects: 100% (4241/4241), 14.74 MiB | 6.69 MiB/s, done.
Resolving deltas: 100% (2590/2590), done.
[Custom Makefile] Copying config file from config.yml to ./scan-io/.
...
[Custom Makefile] Custom Scanio build process complete!
```
