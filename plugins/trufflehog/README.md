# Trufflehog Plugin
The main function of the plugin is to present a top-level interface for a Trufflehog scanner. 

This page is a short plugin description.<br>
You may find additional information in a [scanio-analyse](../../docs/scanio-analyse.md) article.

## Trufflehog command
By defaul a plugin use this command:<br>
```trufflehog + <AdditionalArgs> + [--config <ConfigPath>] + [--<ReportFormat>] + --no-verification + filesystem + <RepoPath>```<br>

Where:
* AdditionalArgs is additional arguments in ```scanio analyse``` command after ```--```.
* ConfigPath is a path to a custom config. Will be applied if you use ```--config``` or ```-c``` in ```scanio analyse``` command.
* ReportFormat is a non-default format of a report. trufflehog supports only json. Will be applied if you use ```--format``` or ```-o```. 
* ```--no-verification``` it means that all found secrets will not be validated by using external systems. 
* ```filesystem``` is a command in terms of Trufflehog. It means the scanner will be searching secrets on a local file system. 
* RepoPath is a local path to a file or a folder with code. 

## Installing Dependencies
If you build the Scanio code using Docker or pull a pre-built container, you do not need to separately install the Trufflehog dependencies. However, if you build the Scanio code to a binary, you will need to install Trufflehog before using the application.
You can refer to the Trufflehog [documentation](https://github.com/trufflesecurity/trufflehog#floppy_disk-installation) for more information.

## Commands
* Analysing using trufflehog with an input file argument.<br>
```scanio analyse --scanner trufflehog --input-file /Users/root/.scanio/output.file --format json -j 2```
* Analysing using trufflehog with a specific path .<br>
```scanio analyse --scanner trufflehog --format json -j 1 /tmp/my_project```
* Analysing using trufflehog with an input file and custom rules.<br>
```scanio analyse --scanner trufflehog --config /Users/root/scan-io-trufflehog-rules --input-file /Users/root/.scanio/output.file --format json -j 2```
* Analysing using trufflehog with an input file and additional arguments.<br>
```scanio analyse --scanner trufflehog --input-file /Users/root/.scanio/output.file --format json -- --debug```

## Results of the Command
If you use an **input file** argument the command will save results into a home directory: ```~/.scanio/results/+<VCSURL>+<Namespace>+<repo_name>/<scanner_name>.<report_format>```.<br><br>

If you use a specific **path** argument the command will save results into the same directory:<br>
* ```scanio analyse --scanner <scanner_name> --format <report_format> /tmp/my_project```
* Result path - ```/tmp/my_project/<scanner_name>.<report_format>```r>

