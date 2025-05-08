# Reference

This section provides detailed technical documentation for Scanio’s commands, configurations, plugins, etc. It serves as a definitive guide for advanced users and developers. The Reference section is for users who need in-depth technical details and specifications.

## Articles
### Common
<!-- - [Scanio Basics](scanio.md): High-level overview of Scanio’s architecture, features, and core concepts. -->
- [Configuration](configuration.md): Detailed documentation of core and plugin configuration options.

### Commands
- [List Command](cmd-list.md): Describes repository discovery functionality across supported VCS platforms, available filtering options, and command output structure.
- [Fetch Command](cmd-fetch.md): Explains repository fetching logic, supported authentication types, URL formats, and command output structure.
- [Analyse Command](cmd-analyse.md): Provides details on running security scanners, handling input data, configuring output formats, and command output structure.
- [To-HTML Command](cmd-to-html.md): Explains conversion of SARIF reports to human-friendly HTML format, code snippet inclusion, and template customization options.
- [Report Patch Command](cmd-report-patch.md): Details how to make structured modifications to SARIF reports, including different filtering capabilities and actions.

### Plugins
- [Bitbucket Plugin](plugin-bitbucket.md): Plugin-specific usage instructions, authentication setup, supported actions, and URL formats for Bitbucket.
- [GitHub Plugin](plugin-github.md): Plugin-specific usage instructions, authentication setup, supported actions, and URL formats for GitHub.
- [GitLab Plugin](plugin-gitlab.md): Plugin-specific usage instructions, authentication setup, supported actions, and URL formats for GitLab.
- [Semgrep Plugin](plugin-semgrep.md): Plugin-specific documentation for the Semgrep scanner plugin, configuration, arguments, and usage examples.
- [Trufflehog Plugin](plugin-trufflehog.md): Plugin-specific documentation for the Trufflehog scanner plugin, configuration, arguments, and usage examples.
- [CodeQL Plugin](plugin-codeql.md): Plugin-specific documentation for the CodeQL scanner plugin, configuration, arguments, and usage examples.
- [Bandit Plugin](plugin-Bandit.md): Plugin-specific documentation for the Bandit scanner plugin, configuration, arguments, and usage examples.
- [Trufflehog3 Plugin](plugin-trufflehog3.md): Plugin-specific documentation for the Trufflehog3 scanner plugin, configuration, arguments, and usage examples.

### Scripts
- [Makefile](makefile.md): Provides details regarding automates building, cleaning, and managing artifacts locally for the Scanio CLI core binary, plugin binaries, Docker image and python environment for rule folder compile.
- [Makefile for Custom Build](makefile-custom-build.md): This page describes the available targets and variables in the [`Makefile`](../../scripts/custom-build/Makefile). This [`Makefile`](../../scripts/custom-build/Makefile) supports custom deployments of Scanio, including cases where users have their own versions of Scanio, plugins, and custom rule sets. 
- [Rules Set Builder](rules-set-builder.md): This page describes a [rules.py](../../scripts/rules/README.md) Python script which automates the process of building rule sets for the Scanio Orchestrator based on a YAML configuration file.
