# Semgrep plugin
The main function of the plugin is to present a top-level interface for a semgrep scanner. 

This page is a short plugin description.<br>
You may find additional information in a [scanio-analyse](../../docs/scanio-analyse.md) articles.<br><br>

## Commands
* Analysing from an input file.<br>
```scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif```
* Analysing from an input file with custom rules.<br>
```scanio analyse --scanner semgrep --config /Users/root/scan-io-semgrep-rules --input-file /Users/root/.scanio/output.file --format sarif```
* Analysing from an input file with additional agruments.<br>
```scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif --args "--verbose,--severity,INFO"```<br><br>

## Results of the command
The command is saving results into a home directory ```~/.scanio/results/+<VCSURL>+<Namespace>+<repo_name>/<scanner_name>.<report_format>```.<br><br>
