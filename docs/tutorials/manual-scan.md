# Manual Scan Tutorial

This tutorial guides you through scanning a project using the Scanio CLI, with [OWASP's Juice Shop](https://github.com/juice-shop/juice-shop) as the example project.

## Prerequisites

Before you start, make sure you have the following installed on your system:
* Scanio CLI and plugins;
* [Semgrep](https://semgrep.dev/docs/getting-started/quickstart/).

**Note:** Scanio operates over existing scanning tools, such as Semgrep.
If you intend to use a specific scanning tool, make sure it's installed before running Scanio. Refer to the tool's installation instructions for guidance.

## Step 1. Clone the Repository

Clone a repository using tools you are familiar with or use the Scanio CLI's built-in `fetch` command. Here's an example using Scanio CLI: 
```
scanio fetch --vcs github --auth-type http https://github.com/juice-shop/juice-shop
```  
Upon completion, you will see the target folder where the project was cloned, like:  
```
...
[INFO]  plugin-vcs.github: A fetch function finished: branch= repo=juice-shop targetFolder=~/.scanio/projects/github.com/juice-shop/juice-shop timestamp=2023-12-25T17:10:32.788+0100
...
```
The `fetch` command preserves the VCS URL, path structure, and adds a prefix `~/.scanio/projects`.

## Step 2. Scan the Repository

To analyze the project, use the `analyze` command with Scanio CLI. Specify the scanner name and the path to your project:  
```
scanio analyse --scanner semgrep ~/.scanio/projects/github.com/juice-shop/juice-shop
```  

If there are no mistakes, the Scanio CLI will output the target path of the resulting file: 
```
...
[INFO]  plugin-scanner.semgrep: Result is saved to: path to a result file=~/.scanio/projects/github.com/juice-shop/juice-shop/semgrep-2023-12-25T16:18:23Z.raw timestamp=2023-12-25T17:22:24.129+0100
...
```  

By default, the output will be in the default format for a tool. You can start reviewing results by opening files:  
`cat ~/.scanio/projects/github.com/juice-shop/juice-shop/semgrep-2023-12-25T16:18:23Z.raw`
