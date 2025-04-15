# Trufflehog3 Plugin

The Trufflehog3 plugin provides integration with the [Trufflehog3 scanner](https://github.com/feeltheajf/trufflehog3) within Scanio. It enables flexible execution of Trufflehog3 scans as part of CI/CD workflows or manual security audits.

This plugin supports analyzing single projects or multiple repositories (via input from the `list` command), allowing configuration customization and fine-tuning scan execution with Trufflehog3-specific arguments.

## Table of Contents
- [Supported Actions](#supported-actions)
- [Plugin Dependencies](#plugin-dependencies)
- [Trufflehog3 Scan Execution Logic](#trufflehog3-scan-execution-logic)
  - [How Scanio Builds the Trufflehog3 Command](#how-scanio-builds-the-trufflehog3-command)
    - [Additional Arguments](#additional-arguments)  
  - [Trufflehog3 Configuration File](#trufflehog3-configuration-file)
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
| Trufflehog3 Configuration Support  | ✅                              |
| Output Formats (Hard Validation) | JSON, Text, HTML. Default: JSON   |
| Output Formats Conversion | SARIF, Markdown                          |


## Plugin Dependencies
If you use Scanio with Docker (building locally or using a pre-built image), all required Trufflehog3 dependencies are included.

However, if you build Scanio as a standalone binary (without Docker), you must install Trufflehog3 separately on your system to use this plugin.

Refer to the official [Trufflehog3 Intallation Guide](https://github.com/feeltheajf/trufflehog3?tab=readme-ov-file#installation) for installation instructions.

## Trufflehog3 Scan Execution Logic
The Trufflehog3 plugin applies passed arguments to the Trufflehog3 call. For detailed information about Trufflehog3's CLI capabilities, refer to the [official documentation](https://github.com/feeltheajf/trufflehog3?tab=readme-ov-file#usage).

### How Scanio Builds the Trufflehog3 Command
When running the analyse command with the Trufflehog3 plugin, Scanio automatically translates the provided arguments into corresponding Trufflehog3 CLI flags.

Running the following Scanio command:
```bash
scanio analyse --scanner trufflehog3 --format json --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project
```
Will execute the following Trufflehog3 command internally:
```bash
trufflehog3 --rules /path/to/scanner-config --format sarif -z --output /path/to/scanner_results /path/to/my_project
```

The `-z` flag forces Trufflehog3 to always exit with a zero status code. We use this option because Trufflehog3 may return a non-zero exit code even when the scan completes successfully without any errors.

### Additional Arguments 
Scanio allows users to pass any extra Trufflehog3 arguments directly, all args after `--` will be added to the Bandit command as is. These arguments will be inserted before the default Scanio-generated arguments, allowing users to override defaults if necessary.

```bash
scanio analyse --scanner trufflehog3 --format json --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project -- --no-history

This command will internally run:
```bash
trufflehog3  --no-history --rules /path/to/scanner-config --format sarif -z --output /path/to/scanner_results /path/to/my_project
```

### Trufflehog3 Configuration File

The Scanio Trufflehog3 plugin supports flexible configuration using flags or predefined config.

You can configure Trufflehog3 scans provide a local path to your custom config.
```bash
scanio analyse --scanner trufflehog3 --format json --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project
```
> [!TIP]
> In the Docker container the rules for Trufflehog3 are placed in `/scanio/rules/trufflehog3/<config_name>`

**Default Configuration Behavior**<br>
If no configuration is explicitly provided via the `--config/c` flag, the flag is ignored.

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
The Trufflehog3 plugin performs a soft validation of the specified output format from `--format/f` flag.

> [!WARNING]
> If the provided format is not in the list of officially supported formats, the plugin logs a warning but proceeds with the scan using the given format. The plugin will proceed scanning with the provided format.

If the argument is not provided, will be used default raw format dedicated by the scanner, the result will be saved into the same folder with the code and file with extention `.raw`.

#### Report Format Conversion
The Trufflehog3 plugin supports automatic conversion of scan results to SARIF and Markdown formats. If you specify:
```bash
--format sarif
--format markdown
```
Scanio will internally run Trufflehog3 with the default `json` output format and then convert the results into the specified format(s). Both the original JSON report and the converted formats will be saved to disk.

## Validation
The Trufflehog3 plugin enforces the following validation rules:
- **Args Validation**: Verifying that the target path exists and is accessible for args.
- **Report Format Validation**: Plugin checks if the provided output format is officially supported by the known Trufflehog3 version.

## Usage Examples

### Scan a Single Directory
This action analyzes the source code of a specified repository.
```bash
scanio analyse --scanner trufflehog3 /path/to/my_project
```

### Bulk Scan from Input File
The `analyse` command seamlessly integrates with the `list` command by allowing users to use the output of the `list` command as input for scanning fetched repositories. The `--input-file (-i)` option in the `analyse` command accepts a file generated by the `list` command. The format of the file aligns with the JSON structure documented in the [List Command Output Format](cmd-list.md#command-output-format).

> [!NOTE]  
> This feature is particularly useful for bulk scanning, enabling the orchestrator to spawn multiple scanner instances for scanning multiple codebases in a single operation efficiently.

```bash
scanio analyse --scanner trufflehog3 --input-file /path/to/list_output.file -j 5 /path/to/my_project
```

The `-j` flag defines the number of concurrent scanning threads.

### Optional Arguments

**Report Format**<br> 
Specify the report format using the `--format/-f` flag:
```bash
scanio analyse --scanner trufflehog3 --format json /path/to/my_project
```

**Config File**<br>  
The Scanio Trufflehog3 plugin supports configuration the scanner through the config file or args.

```bash
scanio analyse --scanner trufflehog3 --config /path/to/scanner-config /path/to/my_project
```

**Report File**<br>  
Specify where to save the report:
```bash
scanio analyse --scanner trufflehog3 --format json --output /path/to/scanner_results /path/to/my_project # File name auto-generated
scanio analyse --scanner trufflehog3 --format json --output /path/to/scanner_results.json /path/to/my_project # Results saved to the specified file
```



