# Semgrep Plugin

The Semgrep plugin provides integration with the [Semgrep scanner](https://semgrep.dev/) within Scanio. It enables flexible execution of Semgrep scans as part of CI/CD workflows or manual security audits.

This plugin supports analyzing single projects or multiple repositories (via input from the `list` command), allowing configuration customization and fine-tuning scan execution with Semgrep-specific arguments.

## Table of Contents
- [Supported Actions](#supported-actions)
- [Plugin Dependencies](#plugin-dependencies)
- [Semgrep Scan Execution Logic](#semgrep-scan-execution-logic)
  - [Scan mode (default)](#scan-mode-default)
  - [CI mode (--command ci)](#ci-mode---command-ci)
  - [Additional Arguments](#additional-arguments)
  - [Semgrep Configuration File](#semgrep-configuration-file)
  - [Report File Handling](#report-file-handling)
  - [Report File Formats](#report-file-formats)
- [Validation](#validation)
- [Usage Examples](#usage-examples)
  - [Scan a Single Directory](#scan-a-single-directory)
  - [Bulk Scan from Input File](#bulk-scan-from-input-file)
  - [Optional Arguments](#optional-arguments)
- [Known Issues and Fixes](#known-issues-and-fixes)
  

## Supported Actions
| Feature                              | Supported                          |
|-------------------------------------|------------------------------------|
| Single Project Scanning             | ✅                                  |
| Bulk Scanning via Input File        | ✅                                  |
| Custom Scanner Arguments            | ✅                                  |
| Semgrep Configuration Support       | ✅                                  |
| Semgrep CI mode (`semgrep ci`)      | ✅                                  |
| Output Formats (Soft Validation)    | JSON, JUNIT-XML, SARIF, Text, Vim  |

## Plugin Dependencies
If you use Scanio with Docker (building locally or using a pre-built image), all required Semgrep dependencies are included.

However, if you build Scanio as a standalone binary (without Docker), you must install Semgrep separately on your system to use this plugin.

Refer to the official [Semgrep Getting Started Guide](https://semgrep.dev/docs/getting-started/) for installation instructions.

## Semgrep Scan Execution Logic

The Semgrep plugin supports two execution modes selected via the `--command` flag on `scanio analyse`.

| Mode | `--command` value | Semgrep subcommand | Rules source | Target argument |
|------|------------------|--------------------|--------------|-----------------|
| Scan (default) | _(omitted)_ | `semgrep scan` | `--config` / local rules | explicit path |
| CI | `ci` | `semgrep ci` | Semgrep AppSec Platform | working directory |

### Scan mode (default)

The default mode uses `semgrep scan`. Scanio translates the provided arguments into the corresponding Semgrep CLI flags.

Running the following Scanio command:
```bash
scanio analyse --scanner semgrep --format sarif --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project
```
Executes internally:
```bash
semgrep scan --sarif -f /path/to/scanner-config --metrics=off --output /path/to/scanner_results /path/to/my_project
```

> [!NOTE]  
> Scanio automatically appends `--metrics=off` when a custom configuration is provided to disable Semgrep telemetry.

### CI mode (`--command ci`)

CI mode uses `semgrep ci`, which is designed for use with the [Semgrep AppSec Platform](https://semgrep.dev/products/semgrep-appsec-platform/). In this mode:

- Rules are fetched from the platform using the `SEMGREP_APP_TOKEN` environment variable — the `--config` flag is not supported and is ignored.
- No target path is passed as an argument; Semgrep scans the working directory (set to the value of `--output`'s base directory, or the path argument).
- Exit code 1 from semgrep means findings were found — Scanio treats this as a successful scan, not an error.

Running the following Scanio command:
```bash
scanio analyse --scanner semgrep --command ci --format sarif --output /path/to/scanner_results /path/to/repo_root -- --baseline-commit $BASE_SHA
```
Executes internally (with working directory set to `/path/to/repo_root`):
```bash
semgrep ci --baseline-commit $BASE_SHA --sarif --output /path/to/scanner_results
```

> [!NOTE]
> `SEMGREP_APP_TOKEN` must be set in the environment. Without it, `semgrep ci` will not fetch organisation policies from the platform.

### Additional Arguments 
Scanio allows users to pass any extra Semgrep arguments directly. All args after `--` are appended to the Semgrep command before the Scanio-generated flags.
```bash
scanio analyse --scanner semgrep --format sarif --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project -- --verbose --severity INFO
```

This command will internally run:
```bash
semgrep scan --verbose --severity INFO --sarif -f /path/to/scanner-config --metrics=off --output /path/to/scanner_results /path/to/my_project
```

### Semgrep Configuration File

The Scanio Semgrep plugin supports flexible configuration using custom, predefined or publicly avaliable rule sets.

You can configure Semgrep scans in two ways:
- Provide a local path to your custom rules or rule sets.
```bash
scanio analyse --scanner semgrep --config /path/to/scanner-config /path/to/my_project
```
> [!TIP]
> In the Docker container the rules for semgrep are placed in `/scanio/rules/semgrep/<rule_set_name>`

- Use rules directly from the [Semgrep Registry](https://semgrep.dev/r).
```bash
scanio analyse --scanner semgrep --config p/ci /path/to/my_project
```

**Default Configuration Behavior**<br>
If no configuration is explicitly provided via the `--config/c` flag:
- `p/ci` rule set is used for CI pipelines (when CI mode is enabled).
- `p/default` rule set is used in other cases (when User mode is enabled).

**Disabling Semgrep Telemetry**<br>
If a custom configuration is provided (any value except `auto`), Scanio automatically disables Semgrep telemetry reporting by setting `--metrics=off`.
This prevents scan metadata from being sent to Semgrep's servers.

> [!TIP]
> For more details refer to the official [Semgrep Metric](https://semgrep.dev/docs/metrics)


### Report File Handling
The `--output` flag in the analyse command supports both file and folder paths for storing scan results.

- If a folder is provided → Scanio will generate the report file name automatically and save the report to the provided directory.
- If a file path is provided → The results will be saved directly to the specified file.

```bash
# Output to a folder (auto-generated report file name)
--output /path/to/scanner_results

# Output to a specific file
--output /path/to/scanner_results/report.json
```

Depending on the mode, the report file name is generated differently:
- User Mode: Templates follow the structure `scanio-report-<plugin_name>.<report_ext>`.
- CI Mode: Templates include a timestamp for uniqueness: `scanio-report-<current_time.RFC3339>-<plugin_name>.<report_ext>`.

### Report File Formats
The Semgrep plugin performs a soft validation of the specified output format from `--format/f` flag.

> [!WARNING]
> If the provided format is not in the list of officially supported formats, the plugin logs a warning but proceeds with the scan using the given format. The plugin will proceed scanning with the provided format.

## Validation
The Semgrep plugin enforces the following validation rules:
- **Args Validation**: Verifying that the target path exists and is accessible.
- **Report Format Validation**: Plugin checks if the provided output format is officially supported by the known Semgrep version. An unsupported format logs a warning but does not abort the scan.
- **Command Validation**: Only `ci` is accepted as a `--command` value. Any other non-empty value returns an error before the scan starts.

## Usage Examples

### Scan a Single Directory
This action analyzes the source code of a specified repository.
```bash
scanio analyse --scanner semgrep /path/to/my_project
```

### Bulk Scan from Input File
The `analyse` command seamlessly integrates with the `list` command by allowing users to use the output of the `list` command as input for scanning fetched repositories. The `--input-file (-i)` option in the `analyse` command accepts a file generated by the `list` command. The format of the file aligns with the JSON structure documented in the [List Command Output Format](cmd-list.md#command-output-format).

> [!NOTE]  
> This feature is particularly useful for bulk scanning, enabling the orchestrator to spawn multiple scanner instances for scanning multiple codebases in a single operation efficiently.

```bash
scanio analyse --scanner semgrep --input-file /path/to/list_output.file -j 5
```

The `-j` flag defines the number of concurrent scanning threads.

### Optional Arguments

**Report Format**<br> 
Specify the report format using the `--format/-f` flag:
```bash
scanio analyse --scanner semgrep --format sarif /path/to/my_project
```

**Rule Set Config**<br>  
The Scanio Semgrep plugin supports flexible configuration using custom, predefined, or publicly available rule sets.

```bash
scanio analyse --scanner semgrep --config /path/to/scanner-config /path/to/my_project  # Local rule set
scanio analyse --scanner semgrep --config p/ci /path/to/my_project  # Semgrep registry rule sets
```

**Report File**<br>  
Specify where to save the report:
```bash
scanio analyse --scanner semgrep --output /path/to/scanner_results /path/to/my_project  # File name auto-generated
scanio analyse --scanner semgrep --output /path/to/scanner_results/report.json /path/to/my_project  # Results saved to the specified file
```


## Known Issues and Fixes
### ```Semgrep does not support Linux ARM64"``` 
**Cause**<br>
This error may occur if you are using a Mac with an Apple M-series (ARM64) chip and building the Scanio Docker container locally. By default, Scanio is built for the `linux/amd64` platform.

**Solution**<br>
To resolve this, ensure you build the Docker container explicitly for the `linux/amd64` platform.

Use Scanio's default make command to build the container:
```
make build docker
```

Or build it manually using:
```
docker build --platform linux/amd64 -t scanio .
```

In some cases, you may also need to specify the platform explicitly when running the container:
```
docker run --rm \
           --platform linux/amd64 \
           scanio version
```






