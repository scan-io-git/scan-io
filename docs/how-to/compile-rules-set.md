# How to Compile Custom Rule Sets in Scanio

This guide walks you through the process of compiling custom rule sets using Scanio's rule-building script. It is useful for security engineers and developers who needs to use special rulesets (e.g., Semgrep) to their organizational needs and compliance.

You will:
- Learn how to define your own `scanio_rules.yaml` file for different tools and rule sets
- Use `Makefile` which handles `rules.py` builder script under the hood to fetch and organize selected rules before the build a Docker image


## Table of Contents

- [Overview](#overview)
- [Step 1: Prepare the YAML Configuration File](#step-1-prepare-the-yaml-configuration-file)
- [Step 2: Compile Custom Rule Sets via Makefile](#step-2-compile-custom-rule-sets-via-makefile)
- [Verify the Build](#verify-the-build)
- [What’s Next](#whats-next)


## Overview

Scanio supports built-in and registry-provided rulesets. However, in many real-world scenarios, you may want:
- More targeted rules (specific technologies or risks)
- Reduced false positives
- Faster scan times
- Integration of private/internal rules

Scanio allows you to define these rules in a structured YAML config file and use a builder tool to fetch and organize them.

## Step 1: Prepare the YAML Configuration File

The configuration file is called `scanio_rules.yaml` and follows a specific structure that maps tools (like semgrep) to one or more rule sets per tool.

Example: [scanio_rules.yaml](../../scripts/rules/scanio_rules.yaml)
```yaml
tools:
  semgrep:
    rulesets:
      default:
        - repo: https://github.com/semgrep/semgrep-rules.git
          branch: develop
          paths:
            - go/lang/security/audit/net/bind_all.yaml
            - java/lang/security/audit/crypto/use-of-md5-digest-utils.yaml
        - repo: https://github.com/trailofbits/semgrep-rules.git
          branch: main
          paths:
            - python/pickles-in-numpy.yaml
            - go/hanging-goroutine.yaml
```

Explanation:
- `tools`: Top-level key. You can add multiple tools (e.g., semgrep, trufflehog, etc.).
- `rulesets`: Each scanner can have multiple named rulesets (e.g., default, audit, dev, ci)
- `repo`: Git URL for the repository with rules
- `branch`: Which branch to use
- `paths`: Relative paths in the repo that point to specific `.yaml` rule files

> [!TIP]
> You can maintain your own fork of the rules or refer to official sources like Semgrep Registry and Trail of Bits.

## Step 2: Compile Custom Rule Sets via Makefile

First of all, you should clone the Scanio repository:
```bash
git clone https://github.com/scan-io-git/scan-io
```

Copy your prepared `scanio_rules.yaml` into the root of the repository and jump into the copied repository:
```bash
cp scanio_rules.yaml scan-io/
cd scan-io
```

The project provides auto build of rule sets via [Makefile](../../Makefile), command `make build-rules`. 
Run this command with providing a path to your prapared configuration file :
```bash
make build-rules RULES_CONFIG=scanio_rules.yaml
```

The script install all necessary python dependencies into the virtual environment, clear `rules/` folder, parse the provided yaml config, clone the provided repositories and copy all the listed rules. After building, the script will create the following folder layout:
```bash
rules/
  ├── semgrep/
  │   ├── default/
  │   │   ├── [copied .yaml rules]
  │   │   └── scanio_rules.yaml.back
  └── scanio_rules.yaml
```

- Rules are nested per tool and ruleset.
- Each tool/ruleset directory has a backup of the config.
- A global copy of your original config is saved at the root of rules/.


## What’s Next

Now that your custom rules are built inside `rules/` folder, you can build a Docker image - [How to Build Scanio](build-scanio.md#option-1-build-docker-image).