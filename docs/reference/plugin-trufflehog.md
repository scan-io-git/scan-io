# Trufflehog Plugin

The Trufflehog plugin provides integration with the [Trufflehog scanner](https://github.com/trufflesecurity/trufflehog) within Scanio. It enables flexible execution of Trufflehog scans as part of CI/CD workflows or manual security audits.

This plugin supports analyzing single projects or multiple repositories (via input from the `list` command), allowing configuration customization and fine-tuning scan execution with Trufflehog-specific arguments.

## Table of Contents
- [Supported Actions](#supported-actions)
- [Plugin Dependencies](#plugin-dependencies)
- [Trufflehog Scan Execution Logic](#trufflehog-scan-execution-logic)
  - [How Scanio Builds the Trufflehog Command](#how-scanio-builds-the-trufflehog-command)
    - [Additional Arguments](#additional-arguments)  
  - [Trufflehog Configuration File](#trufflehog-configuration-file)
  - [Report File Handling](#report-file-handling)
  - [Report File Formats](#report-file-formats)
- [Validation](#validation)
- [Usage Examples](#usage-examples)
  - [Scan a Single Directory](#scan-a-single-directory)
  - [Bulk Scan from Input File](#bulk-scan-from-input-file)
  - [Optional Arguments](#optional-arguments)


## Supported Actions
| Feature                         | Supported                          |
|--------------------------------|-------------------------------------|
| Single Project Scanning        | ✅                                  |
| Bulk Scanning via Input File   | ✅                                  |
| Custom Scanner Arguments       | ✅                                  |
| Trufflehog Configuration Support | ✅                                |
| Output Formats Conversion       | JSON                               |


## Plugin Dependencies
If you use Scanio with Docker (building locally or using a pre-built image), all required Trufflehog dependencies are included.

However, if you build Scanio as a standalone binary (without Docker), you must install Trufflehog separately on your system to use this plugin.

Refer to the official [Trufflehog Intallation Guide](https://github.com/trufflesecurity/trufflehog?tab=readme-ov-file#floppy_disk-installation) for installation instructions.

## Trufflehog Scan Execution Logic
The Trufflehog plugin utilizes the underlying semgrep `filesystem` command for performing scans. For detailed information about Trufflehog's CLI capabilities, refer to the official repository](https://github.com/trufflesecurity/trufflehog).

The `filesystem` command means to [scan individual files or directories](https://github.com/trufflesecurity/trufflehog).

### How Scanio Builds the Trufflehog Command
When running the analyse command with the Trufflehog plugin, Scanio automatically translates the provided arguments into corresponding Trufflehog CLI flags.

Running the following Scanio command:
```bash
scanio analyse --scanner trufflehog --format json --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project
```
Will execute the following Trufflehog command internally:
```bash
trufflehog --config /path/to/scanner-config --json --no-verification filesystem /path/to/my_project
```

The `--no-verification` flag defines the recursive method of scanning. 

### Additional Arguments 

Scanio allows users to pass any extra Trufflehog arguments directly, all args after `--` will be added to the Trufflehog command as is. These arguments will be inserted after the underlying `filesystem` command. [Issue](https://github.com/scan-io-git/scan-io/issues/86) with the reason why.

```bash
scanio analyse --scanner trufflehog --format json --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project -- -x .git
```

This command will internally run:
```bash
trufflehog --config /path/to/scanner-config --json --no-verification filesystem -x .git /path/to/my_project
```

### Trufflehog Configuration File

The Scanio Trufflehog plugin supports flexible configuration using flags or predefined config.

You can configure Trufflehog scans provide a local path to your custom config.
```bash
scanio analyse --scanner trufflehog --format json --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project
```
> [!TIP]
> In the Docker container the rules for Trufflehog are placed in `/scanio/rules/trufflehog/<config_name>`

**Default Configuration Behavior**<br>
If no configuration is explicitly provided via the `--config/c` flag, the flag is ignored.

### Report File Handling
The scanner doesn't support writing a report on a disk, only to the stdout. The Trufflehog plugin handels the output data and write it on a disk.

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
The Trufflehog plugin performs a soft validation of the specified output format from `--format/f` flag.

> [!WARNING]
> If the provided format is not in the list of officially supported formats, the plugin logs a warning but proceeds with the scan using the given format. The plugin will proceed scanning with the provided format.

If the argument is not provided, will be used default raw format dedicated by the scanner, the result will be saved into the same folder with the code and file with extention `.raw`.

## Validation
The Trufflehog plugin enforces the following validation rules:
- **Args Validation**: Verifying that the target path exists and is accessible for args.
- **Report Format Validation**: Plugin checks if the provided output format is officially supported by the known Trufflehog version.

## Usage Examples

### Scan a Single Directory
This action analyzes the source code of a specified repository.
```bash
scanio analyse --scanner trufflehog /path/to/my_project
```

### Bulk Scan from Input File
The `analyse` command seamlessly integrates with the `list` command by allowing users to use the output of the `list` command as input for scanning fetched repositories. The `--input-file (-i)` option in the `analyse` command accepts a file generated by the `list` command. The format of the file aligns with the JSON structure documented in the [List Command Output Format](cmd-list.md#command-output-format).

> [!NOTE]  
> This feature is particularly useful for bulk scanning, enabling the orchestrator to spawn multiple scanner instances for scanning multiple codebases in a single operation efficiently.

```bash
scanio analyse --scanner trufflehog --input-file /path/to/list_output.file -j 5 /path/to/my_project
```

The `-j` flag defines the number of concurrent scanning threads.

### Optional Arguments

**Report Format**<br> 
Specify the report format using the `--format/-f` flag:
```bash
scanio analyse --scanner trufflehog --format json /path/to/my_project
```

**Config File**<br>  
The Scanio Trufflehog plugin supports configuration the scanner through the config file or args.

```bash
scanio analyse --scanner trufflehog --config /path/to/scanner-config /path/to/my_project
```

**Report File**<br>  
Specify where to save the report:
```bash
scanio analyse --scanner trufflehog --format json --output /path/to/scanner_results /path/to/my_project # File name auto-generated
scanio analyse --scanner trufflehog --format json --output /path/to/scanner_results.json /path/to/my_project # Results saved to the specified file
```