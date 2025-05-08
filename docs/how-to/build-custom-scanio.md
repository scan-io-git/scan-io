# How to Build a Custom Installation of Scanio

This guide shows how to build a customized Docker image of Scanio, tailored to your internal needs. This includes:
- Integrating custom rule sets and configuration.
- Using your own Scanio source code or fork.
- Optionally adding internal plugins.

This guide is intended for developers and security engineers who want full control over how Scanio is packaged and deployed — especially in environments without CI pipelines or with custom compliance requirements.


## Table of Contents

- [Overview](#overview)
- [What You'll Need](#what-youll-need)
- [Prepare Your Custom Build Directory/Repository](#prepare-your-custom-build-directoryrepository)
- [Step-by-Step: Build Your Custom Scanio Image](#step-by-step-build-your-custom-scanio-image)
- [Verify the Build](#verify-the-build)
- [What’s Next](#whats-next)

## Overview

To suport custom cases of building the Scanio from local disk or internal company repository we have created a special [Makefile](../../scripts/custom-build/Makefile).

The `Makefile` provided in the Scanio project automates the process of building a Docker image with your configuration, rules, and optionally, source code.
This is a user-focused workflow, ideal for Security teams maintaining custom rules, Scanio plugins and configuration of the multitool.

The workflow automates:
- Cloning your specified version of the Scanio repository.
- Replacing default config and rule definitions.
- Compiling rule sets.
- Building a Docker image.
- Pushing the image to a Docker registry.

You can control most behaviors via command-line variables.

## What You'll Need

Install the following on your machine:
- Docker
- Git
- Make
- Python 3 (used internally to build rule sets)


## Prepare Your Custom Build Directory/Repository

To create a custom build, your working directory should include the following:
- `Makefile` — Orchestration script for building and pushing.
- `config.yml` — Your customized core Scanio configuration.
- `scanio_rules.yaml` — A rule set build configuration file.

These files are sourced from [scripts/custom-build](../../scripts/custom-build/) directory. You may keep these files in the separated internal repository.


## Step-by-Step: Build Your Custom Scanio Image

1. Clone Your Build Directory
```bash
git clone https://my.internal.git/security/scanio-custom-build
cd scanio-custom-build
```

2. Configure Your Inputs

Update the `config.yml` and `scanio_rules.yaml` if needed to match your internal use case . These files will override the default versions in the cloned Scanio repository.


3. Run the Build

Run the following command:
```bash
make build SCANIO_REPO=https://my.internal.git/security/scanio-custom-code VERSION=1.0 REGISTRY=my.registry.com/scanio PLUGINS="github gitlab bitbucket semgrep bandit trufflehog"
```

This command performs:
- Cloning Scanio source from `SCANIO_REPO` (cmd: `clone-scanio-repo`). The arguments should refer to your internal version of Scanio or to the [official repository](https://github.com/scan-io-git/scan-io). 
- Replacing config and rules (cmd: `copy-config`, `copy-rules`)
- Compiling rule sets (cmd: `build-rules`)
- Building a Docker image (cmd: `build-docker`)
- Pushing the image if `REGISTRY` is specified (cmd: `push-docker`)


## Verify the Build

Once the image is built, verify it locally:
```bash
docker run --rm my.registry.com/scanio:1.0 version
```

**Sample output:**
```bash
Core Version: v0.1.2
Plugin Versions:
  github: v0.0.2 (Type: vcs)
  gitlab: v0.0.2 (Type: vcs)
  semgrep: v0.0.2 (Type: scanner)
Go Version: go1.22
Build Time: 2024-04-01T10:00:00Z
```

## What’s Next

For advanced custom building, check:
- [Reference: Makefile for Custom Build](../reference/makefile-custom-build.md)
- [Reference: Makefile](../reference/makefile.md)

For advanced rules, learn more about rule compilation:
- [Reference: Rule Set Builder](../reference/rule-set-builder.md)
