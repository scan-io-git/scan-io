# How to Troubleshoot

When using Scanio, you may encounter issues. This section outlines common troubleshooting steps for resolving issues when running Scanio.


## Table of Contents

- [Logs](#logs)
- [Using Debug Mode](#using-debug-mode)
  - [Example: Running in Debug Mode via Docker](#example-running-in-debug-mode-via-docker)
  - [Example: Running in Debug Mode for Go CLI](#example-running-in-debug-mode-for-go-cli)
- [Command Metadata Files](#command-metadata-files)
- [Using Docker Interactive Bash Mode ](#using-docker-interactive-bash-mode)

## Logs

Logs are stored in `{home_folder}/log` by default.

## Using Debug Mode

By default, Scanio operates at the `INFO` logging level. For more detailed logs, you can increase the verbosity by setting the `SCANIO_LOG_LEVEL` environment variable to `DEBUG`.

This can be particularly useful for investigating scanner execution errors, plugin issues, or configuration problems.

### Example: Running in Debug Mode via Docker
The following command runs the analyse command with Semgrep in debug mode:
```bash
docker run --rm \
    -e SCANIO_LOG_LEVEL=DEBUG \
    -v "/Users/root/development:/data" \
    scanio analyse \
        --scanner semgrep \
        -f sarif \
        -c /scanio/rules/semgrep/developer \
        /data/your-repo-path/
```
Flags explanation:
```
-e SCANIO_LOG_LEVEL=DEBUG \                     # Enables debug logging by passing the env variable
-v "/Users/root/development:/data" \            # Mounts your local directory into the container
--scanner semgrep \                             # Specifies the scanner
-f sarif \                                      # Sets the output format to SARIF
-c /scanio/rules/semgrep/developer \            # Uses a predefined rule set from the container
/data/your-repo-path/                           # Targets the codebase inside the mounted folder
```

### Example: Running in Debug Mode for Go CLI
The following command runs the analyse command with Semgrep in debug mode:
```bash
export SCANIO_LOG_LEVEL=DEBUG
scanio analyse --scanner semgrep -f sarif -c /scanio/rules/semgrep/developer /data/your-repo-path/
```

## Command Metadata Files

Scanio generates structured metadata files that capture information about each command executed and writes it in CI mode in a dedicated `<scanio_home>/artifacts` folder.

Each metadata file contains:
```json
{
  "launches": [
    {
      "args": {
        "<key>": "<value>"  // Arguments passed to the command
      },
      "result": {
        "<key>": "<value>"  // Output from the command
      },
      "status": "OK or FAILED",
      "message": "Error message if any"
    }
  ]
}
```

These files are stored at `/scanio` inside the Docker container and in `~/.scanio` for Go CLI. Filenames follow the format:
```bash
<command>_<plugin>_<timestamp>.scanio-artifact.json
```

For example:
```bash
fetch_github_2025-09-15T09:21:20Z.scanio-artifact.json 
```

## Using Docker Interactive Bash Mode 

You can troubleshoot interactively by opening a shell inside the Scanio container. 

```bash
docker run -it --entrypoint="" \
    -v "/Users/root/development:/data" \ 
    scanio /bin/bash
```

Flags explanation:
```
-it                                        # -i (interactive) and -t (allocate a pseudo-TTY), this makes the container session interactive
--entrypoint=""                            # Overrides the default entrypoint of the Docker image (which is the Scanio CLI).
-v "/Users/root/development:/data" \       # Mounts your local directory into the container
/bin/bash                                  # Custom command to spawn a shell
```

Once inside the container, you can run commands like:

```bash
scanio analyse --scanner semgrep -c /scanio/rules/semgrep/developer /data/your-repo-path/
```

Scan results of the command will appear at:
- Inside the container:  
  `/data/your-repo-path/scanio-report-semgrep-<timestamp>.raw`
- On the host system:  
  `/Users/root/development/your-repo-path/scanio-report-semgrep-<timestamp>.raw`
