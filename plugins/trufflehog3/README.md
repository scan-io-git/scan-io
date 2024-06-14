# Trufflehog3 Plugin
The main function of the plugin is to present a top-level interface for a [Trufflehog3](https://github.com/feeltheajf/trufflehog3) scanner. 

This page is a short plugin description.<br>
You may find additional information in a [scanio-analyse](../../docs/scanio-analyse.md) article.

## Trufflehog3 command
By defaul a plugin use this command:<br>
```trufflehog3 + <AdditionalArgs> + [--rules <ConfigPath>] + [--format <ReportFormat>] + -z + --output <ResultsPath> + <RepoPath>```<br>

Where:
* AdditionalArgs is additional arguments in ```scanio analyse``` command after ```--```.
* ConfigPath is a path to a custom config. Will be applied if you use ```--config``` or ```-c``` in ```scanio analyse``` command.
* ReportFormat is a non-default format of a report. Will be applied if you use ```--format``` or ```-f```. 
* ```-z``` says to Trufflehog3 always exit with zero status code. WE use it because Trufflehog3 sends a not correct exit code even when it finished without errors. 
* ```--output``` is a path on a local disk to file with results. Trufflehog3 will create or rewrite a file with results by using this path.

## Installing Dependencies
If you build the Scanio code using Docker or pull a pre-built container, you do not need to separately install the Trufflehog3 dependencies. However, if you build the Scanio code to a binary, you will need to install Trufflehog3 before using the application.
You can refer to the Trufflehog3 [documentation](https://github.com/feeltheajf/trufflehog3#installation) for more information.

## Commands
* Analysing using trufflehog3 with an input file argument.<br>
```scanio analyse --scanner trufflehog3 --input-file /Users/root/.scanio/output.file --format json -j 2```
* Analysing using trufflehog3 with a specific path .<br>
```scanio analyse --scanner trufflehog3 --format json -j 1 /tmp/my_project```
* Analysing using trufflehog3 with an input file and custom rules.<br>
```scanio analyse --scanner trufflehog3 --config /Users/root/scan-io-trufflehog-rules --input-file /Users/root/.scanio/output.file --format json -j 2```
* Analysing using trufflehog3 with an input file and additional arguments.<br>
```scanio analyse --scanner trufflehog3 --input-file /Users/root/.scanio/output.file --format json -- --vvv```

## Results of the Command
If you use an **input file** argument the command will save results into a home directory: ```~/.scanio/results/+<VCSURL>+<Namespace>+<repo_name>/<scanner_name>.<report_format>```.<br><br>

If you use a specific **path** argument the command will save results into the same directory:<br>
* ```scanio analyse --scanner <scanner_name> --format <report_format> /tmp/my_project```
* Result path - ```/tmp/my_project/<scanner_name>.<report_format>```

