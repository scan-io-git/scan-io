# Rule Set Builder

This page describes a [rules.py](../../scripts/rules/README.md) Python script which automates the process of building rule sets for the Scanio Orchestrator based on a YAML configuration file. The script is designed to:
- Parse a given YAML configuration (`scanio_rules.yaml`).
- Clone multiple repositories as defined in a YAML configuration file.
- Copy specific files from the repositories into structured folders within a rules directory.
- Supports interactive and non-interactive directory cleanup.

## Table of Contents

- [Requirements](#requirements)
- [Syntax](#syntax)
- [Command-line Arguments](#command-line-arguments)
- [YAML Configuration File](#yaml-configuration-file)
    - [Structure](#structure)
- [Structure Creation for Copied Files](#structure-creation-for-copied-files)
- [Usage](#usage)


## Requirements
- Python 3.x
- `pyyaml` for YAML parsing.
- `gitpython` for repository cloning.
- `colorama` for colored terminal output.
- `tqdm` for progress bars.

All Python dependencies for the script are placed in [requirements.txt](../../scripts/rules/requirements.txt) file. You can install the dependencies using pip3:
```bash
pip3 install -r requirements.txt
```

## Syntax
```bash
python rules.py [-h] [-r RULES] [-f] [--rules-dir RULES_DIR] [-v] [--no-color]
```

## Command-line Arguments

| Option             | Description                                                                                              | Default                    |
|--------------------|----------------------------------------------------------------------------------------------------------|----------------------------|
| `--help`           | Displays help.                                                                                           |  N/A                      |
| `-r, --rules`      | Path to the YAML configuration file.                                                                     | `scanio_rules.yaml` in the script directory |
| `-f, --force`      | Forcefully clean the `rules` directory without confirmation.                                             | N/A                        |
| `--rules-dir`      | Directory where rule sets will be stored.                                                                | `./rules`                  |
| `-v, --verbose`    | Increase verbosity level. Use multiple times for more verbosity: `-v`, `-vv`, `-vvv`.                    | No verbosity               |
| `--no-color`       | Disable colored terminal output.                                                                         | Color enabled by default   |


## YAML Configuration File

The `scanio_rules.yaml` file defines the tools, rulesets, and repositories to process. You can define multiple tools and rulesets in the configuration file, each specifying the repositories to clone and the file paths to copy.

Here's an example structure:

```yaml
tools:
  tool_name:
    rulesets:
      ruleset_name:
        - repo: https://github.com/user/repo.git
          branch: main
          paths:
            - path/to/file1
            - path/to/file2
```

### Structure
- `tools`: A dictionary containing the tool name as the key.
- `rulesets`: A list of rulesets for each tool.
- `repo`: The URL of the repository to clone.
- `branch`: The branch to clone.
- `paths`: A list of file paths to copy from the cloned repository.

## Structure Creation for Copied Files
The script creates a structured folder layout inside the rules directory for each tool and its corresponding rulesets as defined in the YAML configuration file. For each repository, it clones the repository, navigates through the paths specified in the configuration, and copies the defined files to a corresponding path within the ruleset's folder.

For instance, for a tool named semgrep and a ruleset default, a directory structure like this will be created:
```bash
rules/
  └── semgrep/
      └── default/
          └── [copied_files_here]
```

For each tool, a `scanio_rules.yaml.back` file will be created in its corresponding rule set folder. In this example, it will be located at `rules/semgrep/default/`.

Additionally, the provided `scanio_rules.yaml` configuration file will be added to the root `rules/` directory. This file illustrates how the current structure was built.

## Usage

**Basic usage** with a YAML configuration:
```bash
python rule_set_builder.py
```
    
You may specify:
- A specific path to the config file: `--rules /path/to/scanio_rules.yaml`
- Force clean the `rules` directory before running the script: `--force`
- Verbose mode: `-v`, `-vv`,  `-vvv`
- Color mode, to turn off: use `--no-color`

**Sample output:**
```bash
Using temporary directory: /tmp/tmp5df4h7jc
Processing tool: example_tool
    Processing ruleset: example_ruleset
    Cloning https://github.com/user/repo (branch: main) into /tmp/tmp5df4h7jc
      Processing: 100%|████████████████████████| 3/3 [00:01<00:00,  2.12files/s]
    Finished processing ruleset: example_ruleset
Finished processing tool: example_tool

Temporary directory cleaned up automatically.
```
