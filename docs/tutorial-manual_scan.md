# Tutorial for a manual scan

In this tutorial, we will learn how to scan a project with scanio cli.  
As an experimental project we will use [OWASP's juice-shop](https://github.com/juice-shop/juice-shop).

## Prerequisites

In this lesson we are going to analyze a project from github using semgrep.  
Make sure to get scanio cli and plugins installed in your system.
Make sure to [install semgrep](https://semgrep.dev/docs/getting-started/quickstart/). For examples with `pip install semgrep`. To be able to run any other scanning tool, you have to install it before running scanio. You should follow specific tool installation instructions for that.  

## Step 1. Clone the repo

To clone a repo, you can use tools you are used to. Or we can go with a built-in command `fetch`. This is how it would look like:  
`scanio fetch --vcs github --auth-type http https://github.com/juice-shop/juice-shop`  
In the output, you will see the target folder, where the project was cloned to, like:  
```
...
[INFO]  plugin-vcs.github: A fetch function finished: branch= repo=juice-shop targetFolder=~/.scanio/projects/github.com/juice-shop/juice-shop timestamp=2023-12-25T17:10:32.788+0100
...
```
The scanio cli preserves the vcs url, path structure and adds a prefix `~/.scanio/projects` as a clone to the folder.

## Step 2. Scan the repo

To be able to run any scanning tool, you have to install it before running a scanio analyze command. You should follow specific tool installation instructions for that.  
In this tutorial, we will use `semgrep` as a scanning tool. Let's install it with `pip install semgrep` if it was not installed on your system yet.  

To run the analysis, we will use `analyze` command. The command requires to specify a scanner name and path to your project:  
`scanio analyse --scanner semgrep ~/.scanio/projects/github.com/juice-shop/juice-shop`  

If there are no mistakes, scanio cli will output the target path of the resulting file:  
```
...
[INFO]  plugin-scanner.semgrep: Result is saved to: path to a result file=~/.scanio/projects/github.com/juice-shop/juice-shop/semgrep-2023-12-25T16:18:23Z.raw timestamp=2023-12-25T17:22:24.129+0100
...
```  

By default, the output will be in default for a tool format. Usually, we can start reviewing results by just opening files:  
`cat ~/.scanio/projects/github.com/juice-shop/juice-shop/semgrep-2023-12-25T16:18:23Z.raw`
