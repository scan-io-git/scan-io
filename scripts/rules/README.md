# Rule Set Builder

This Python script automates the process of building rule sets for the Scanio Orchestrator based on a YAML configuration file. The script is designed to:
- Parse a given YAML configuration (`scanio_rules.yaml`).
- Clone repositories defined in the YAML configuration.
- Copy specific files from the repositories into structured folders within a rules directory.

## Features
- Clone multiple repositories as defined in a YAML configuration file.
- Copy specific files from cloned repositories into a target directory.
- Supports interactive and non-interactive directory cleanup.

## Requirements
- Python 3.x
- `colorama` for colored terminal output.
- `tqdm` for progress bars.
- `pyyaml` for YAML parsing.
- `gitpython` for repository cloning.

You can install the dependencies using pip3:
```bash
pip3 install -r requirements.txt
```

## Usage

### Command-line Arguments

| Argument           | Description                                                                                             | Default                    |
|--------------------|---------------------------------------------------------------------------------------------------------|----------------------------|
| `-r, --rules`      | Path to the YAML configuration file.                                                                     | `scanio_rules.yaml` in the script directory |
| `-f, --force`      | Forcefully clean the `rules` directory without confirmation.                                             | N/A                        |
| `--rules-dir`      | Directory where rule sets will be stored.                                                                | `./rules`                  |
| `-v, --verbose`    | Increase verbosity level. Use multiple times for more verbosity: `-v`, `-vv`, `-vvv`.                    | No verbosity               |
| `--no-color`       | Disable colored terminal output.                                                                         | Color enabled by default   |

### Examples

1. **Basic usage** with a YAML configuration:

   ```bash
   python rule_set_builder.py
    ```

2. **Basic usage** with a YAML configuration:
   ```bash
   python rule_set_builder.py --rules /path/to/scanio_rules.yaml
    ```

3. **Force clean the `rules` directory** before running the script:

   ```bash
    python rule_set_builder.py --force
    ```

4. **Run with verbose output**:

   ```bash
   python rule_set_builder.py -vv
   ```

5. **Disable colored output**:

   ```bash
   python rule_set_builder.py --no-color
   ```

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
### Structure:
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


## Example Output

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
