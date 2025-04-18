# Bandit Plugin

The Bandit plugin provides integration with the [Bandit scanner](https://github.com/PyCQA/bandit) within Scanio. It enables flexible execution of Bandit scans as part of CI/CD workflows or manual security audits.

This plugin supports analyzing single projects or multiple repositories (via input from the `list` command), allowing configuration customization and fine-tuning scan execution with Bandit-specific arguments.

## Table of Contents
- [Supported Actions](#supported-actions)
- [Plugin Dependencies](#plugin-dependencies)
- [Bandit Scan Execution Logic](#bandit-scan-execution-logic)
  - [How Scanio Builds the Bandit Command](#how-scanio-builds-the-bandit-command)
    - [Additional Arguments](#additional-arguments)  
  - [Bandit Configuration File](#bandit-configuration-file)
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
| Bandit Configuration Support   | ✅                                  |
| Output Formats (Soft Validation) | CSV, Custom, HTML, TXT, JSON, XML, YAML |


## Plugin Dependencies
If you use Scanio with Docker (building locally or using a pre-built image), all required Bandit dependencies are included.

However, if you build Scanio as a standalone binary (without Docker), you must install Bandit separately on your system to use this plugin.

Refer to the official [Bandit Getting Started Guide](https://bandit.readthedocs.io/en/latest/start.html) for installation instructions.

## Bandit Scan Execution Logic
The Bandit plugin applies passed arguments to the Bandit call. For detailed information about Bandit's CLI capabilities, refer to the [official documentation](https://bandit.readthedocs.io/en/latest/start.html#usage).

### How Scanio Builds the Bandit Command
When running the analyse command with the Bandit plugin, Scanio automatically translates the provided arguments into corresponding Bandit CLI flags.

Running the following Scanio command:
```bash
scanio analyse --scanner bandit --format txt --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project
```
Will execute the following Bandit command internally:
```bash
bandit -f txt -c /path/to/scanner-config -r -o /path/to/scanner_results /path/to/my_project
```

The `-r` flag defines the recursive method of scanning. 

### Additional Arguments 
Scanio allows users to pass any extra Bandit arguments directly, all args after `--` will be added to the Bandit command as is. These arguments will be inserted before the default Scanio-generated arguments, allowing users to override defaults if necessary.
```bash
scanio analyse --scanner bandit --format txt --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project -- --verbose -l
```

This command will internally run:
```bash
bandit --verbose -l -f txt -c /path/to/scanner-config -r -o /path/to/scanner_results /path/to/my_project
```

### Bandit Configuration File

The Scanio Bandit plugin supports flexible configuration using predefined config.

You can configure Bandit scans provide a local path to your custom config.
```bash
scanio analyse --scanner bandit --format txt --config /path/to/scanner-config --output /path/to/scanner_results /path/to/my_project
```
> [!TIP]
> In the Docker container the rules for Bandit are placed in `/scanio/rules/bandit/<config_name>`

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
--output /path/to/scanner_results/report.txt
```

Depending on the mode, the report file name is generated differently:
- User Mode: Templates follow the structure `scanio-report-<plugin_name>.<report_ext>`.
- CI Mode: Templates include a timestamp for uniqueness: `scanio-report-<current_time.RFC3339>-<plugin_name>.<report_ext>`.

### Report File Formats
The Bandit plugin performs a soft validation of the specified output format from `--format/f` flag.

> [!WARNING]
> If the provided format is not in the list of officially supported formats, the plugin logs a warning but proceeds with the scan using the given format. The plugin will proceed scanning with the provided format.

If the argument is not provided, will be used default raw format dedicated by the scanner, the result will be saved into the same folder with the code and file with extention `.raw`.

## Validation
The Bandit plugin enforces the following validation rules:
- **Args Validation**: Verifying that the target path exists and is accessible for args.
- **Report Format Validation**: Plugin checks if the provided output format is officially supported by the known Bandit version.

## Usage Examples

### Scan a Single Directory
This action analyzes the source code of a specified repository.
```bash
scanio analyse --scanner bandit /path/to/my_project
```

### Bulk Scan from Input File
The `analyse` command seamlessly integrates with the `list` command by allowing users to use the output of the `list` command as input for scanning fetched repositories. The `--input-file (-i)` option in the `analyse` command accepts a file generated by the `list` command. The format of the file aligns with the JSON structure documented in the [List Command Output Format](cmd-list.md#command-output-format).

> [!NOTE]  
> This feature is particularly useful for bulk scanning, enabling the orchestrator to spawn multiple scanner instances for scanning multiple codebases in a single operation efficiently.

```bash
scanio analyse --scanner bandit --input-file /path/to/list_output.file -j 5 /path/to/my_project
```

The `-j` flag defines the number of concurrent scanning threads.

### Optional Arguments

**Report Format**<br> 
Specify the report format using the `--format/-f` flag:
```bash
scanio analyse --scanner bandit --format txt /path/to/my_project
```

**Config File**<br>  
The Scanio Bandit plugin supports configuration the scanner through the config file.

```bash
scanio analyse --scanner bandit -config /path/to/scanner-config /path/to/my_project
```

**Report File**<br>  
Specify where to save the report:
```bash
scanio analyse --scanner bandit --format txt --output /path/to/scanner_results /path/to/my_project # File name auto-generated
scanio analyse --scanner bandit --format txt --output /path/to/scanner_results.txt /path/to/my_project  # Results saved to the specified file
```



