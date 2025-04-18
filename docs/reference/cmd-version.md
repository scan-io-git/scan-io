# Version Command
The `version` command provides information about the current version of Scanio core and all available plugins. This command helps to verify which versions of the core application and each plugin are currently being used.


## Table of Contents
- [Syntax](#syntax)
- [Usage Examples](#usage-examples)
- [Command Output Format](#command-output-format)
  - [User Mode Output Format](#user-mode-output-format)
  - [CI Mode Output Format](#ci-mode-output-format)
- [Version Source File](#version-source-file)
- [Validation](#validation)


## Syntax
```bash
scanio version
```

## Usage Example

Print version information in user-friendly format:
```bash
scanio version
```

Force JSON output (only for CI mode, auto-detected):
```bash
SCANIO_CI=true scanio version
```

## Command Output Format

Depending on the mode (CI or User), output format changes:

- In CI mode (detected automatically), output is printed in JSON format.
- In User mode, output is human-readable.

### User Mode Output Format
```bash
Core Version: v0.1.2
Plugin Versions:
  github: v0.0.2 (Type: vcs)
  gitlab: v0.0.2 (Type: vcs)
  semgrep: v0.0.2 (Type: scanner)
Go Version: go1.22
Build Time: 2024-04-01T10:00:00Z
```

### CI Mode Output Format

```json
{
  "versions": {
    "version": "0.1.2",
    "golang_version": "go1.22",
    "build_time": "2024-04-01T10:00:00Z"
  },
  "plugin_details": {
    "github": {
      "version": "0.0.2",
      "plugin_type": "vcs"
    },
    "gitlab": {
      "version": "0.0.2",
      "plugin_type": "vcs"
    },
    "semgrep": {
      "version": "0.0.2",
      "plugin_type": "scanner"
    }
  }
}
```

## Version Source File

Each plugin provides its own `VERSION` file located in the plugin directory. This file must be in JSON format:

```json
{
  "version": "0.0.2",
  "plugin_type": "scanner"
}
```

| Supported Plugin Types           |
|----------------------------|
| scanner     |
| vcs       |

The main `VERSION` file for the Scanio core is located in the root directory of the project. It defines the current version of the tool.
```json
{
  "version": "0.2.0",
}
```

## Validation

- `VERSION` File Presence: Plugin must contain a `VERSION` file to provide version data.
- `VERSION` File Format: `VERSION` file must follow JSON structure.
