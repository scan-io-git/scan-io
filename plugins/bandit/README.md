# Bandit Plugin
The main function of the plugin is to present a top-level interface for a Bandit scanner. 

This page is a short plugin description.<br>
You may find additional information in a [scanio-analyse](../../docs/scanio-analyse.md) article.

## Installing Dependencies
If you build the Scanio code using Docker or pull a pre-built container, you do not need to separately install the Bandit dependencies. However, if you build the Scanio code to a binary, you will need to install Bandit before using the application.
You can refer to the Bandit [documentation](https://bandit.readthedocs.io/en/latest/start.html) for more information.

## Commands
* Analysing using bandit with an input file argument.<br>
```scanio analyse --scanner bandit --input-file /Users/root/.scanio/output.file --format sarif -j 2```
* Analysing using bandit with a specific path .<br>
```scanio analyse --scanner bandit --format json -j 1 /tmp/my_project```
* Analysing using bandit with an input file and custom rules.<br>
```scanio analyse --scanner bandit --config /Users/root/scan-io-bandit-rules --input-file /Users/root/.scanio/output.file --format sarif -j 2```
* Analysing using bandit with an input file and additional arguments.<br>
```scanio analyse --scanner bandit --input-file /Users/root/.scanio/output.file --format sarif -- --verbose --severity INFO```

## Results of the Command
If you use an **input file** argument the command will save results into a home directory: ```~/.scanio/results/+<VCSURL>+<Namespace>+<repo_name>/<scanner_name>.<report_format>```.<br><br>

If you use a specific **path** argument the command will save results into the same directory:<br>
* ```scanio analyse --scanner <scanner_name> --format <report_format> /tmp/my_project```
* Result path - ```/tmp/my_project/<scanner_name>.<report_format>```r>
