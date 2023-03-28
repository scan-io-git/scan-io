# Scanio
Scanio is an application that acts as an orchestrator over plugins. The application is made up of two main parts - the core and plugins.<br>
The core handles the plugins and controls the entire life cycle of the plugins, while the plugins themselves implement work with a variety of functions, other applications, and scanners.<br><br>

## Scenarios for Using Scanio
There are several different scenarios in which you might use Scanio, which are sorted by the environment in which they're used.

### Manual Scanning Process for AppSec Teams and Developers
This scenario involves performing on-demand scanning, which is a common use case for the application. It allows you to manually control any arguments for the scanner that you need, including implementation review scanning, developing custom rules and scanning, scans initiated by developers to self-check, and more.<br><br>

If you're interested in using Scanio for manual scanning, check out the "[Quick Start for a Manual Scanning Process](docs/quick%20start%20for%20a%20manual%20scanning%20process.md)" page!

### Iterative Scanning Process
The main idea behind this approach is to enable constant scanning of a project. This could include regular, iterative scans, or scans with specific rules (such as PCI-DSS code) depending on your requirements.<br><br>

Some of the environments used for this approach include:
- k8s cron jobs.
- VMs/personal devices with ample resources.

<br>

*Quick Start Guid is in progress...*

### Merge Request Scanning Process
The primary objective of this approach is to integrate the orchestrator with CI/CD pipelines and trigger scans automatically after certain actions in your Version Control System (VCS).<br><br>

Some of the environments used for this approach include:
- Tools like Jenkins
- Native VCS CI systems like GitLab CI.

<br>

*Quick Start Guid is in progress...*

## Installation
### Docker Building from Source Code
To build from the source code using Docker, use the following command:
```
make docker
```

<br>

Alternatively, you can use the following command to build a Docker image:
```
docker build -t scanio .
```

### Building the CLI from Source Code
To build the CLI from the source code, use the following command:
```
make build
```

## Articles to Read
Here are some articles that provide more information about using the Scanio application.

### Commands 
These articles cover the different commands available in the application:
* [List](docs/scanio-list.md).
* [Fetch](docs/scanio-fetch.md).
* [Analyse](docs/scanio-analyse.md).

### Plugins
These articles cover the different plugins supported by the application:
* [Bitbucket](plugins/bitbucket/README.md).
* [Semgrep](plugins/semgrep/README.md).