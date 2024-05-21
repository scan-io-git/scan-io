# Scanio Analyse Command
The main function is to present a top-level interface for a specified scanner. The function is handling scanners plugins. <br><br>

|Scanners|Semgrep|Bandit|
|----|-----|---|
|Inherited args|Supported|Not supported|
|Custom config for rules|Supported|Not supported|
|Report format arg|Supported|Not supported|
|Local multithread|Supported|Supported|

## Args of the Command
- "scanner" is the plugin name of the scanner used. The default is semgrep.
- "input-file" or "i" is a file in scanio format with a list of repositories to analyse. The list command could prepare this file.
- "format" or "f" is a format for a report with results. 
- "config" or "c" is a path or type of config for a scanner. The value depends on a particular scanner's used formats. The default is auto. 
- "threads" or "j" is a number of concurrent goroutines. The default is 1.<br><br>

Instead of using an **input file** flag you could use a specific **path** that points to a folder with your code. Check the [link](#analysing-only-one-repository-manually-by-path). <br><br>

Also you are able to add additional arguments to a command. If you want to execute scanner with custom arguments, you could use two dashes (--) to separate additional flags/arguments:<br>
```scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif -j 1 -- --verbose --severity INFO```

## Using Scenarios 
When developing, we aimed at the fact that the program will be used primarily for automation purposes but you still able to use it manually from CLI.

### Analysing from an Input File
The command uses an output format of a List command for analysing required repositories.<br>

If you use an **input file** argument the command will save results into a home directory - ```~/.scanio/results/+<VCSURL>+<Namespace>+<repo_name>/<scanner_name>.<report_format>```.<br>
You can redifine a home directory by using **SCANIO_HOME** environment variable.

#### Semgrep
* Analysing using semgrep with an input file argument.<br>
```scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif -j 2```
* Analysing using semgrep with a specific path .<br>
```scanio analyse --scanner semgrep --format sarif -j 1 /tmp/my_project```
* Analysing using semgrep with an input file and custom rules.<br>
```scanio analyse --scanner semgrep --config /Users/root/scan-io-semgrep-rules --input-file /Users/root/.scanio/output.file --format sarif -j 2```
* Analysing using semgrep with an input file and additional arguments.<br>
```scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif -- --verbose --severity INFO```

#### Bandit
* Analysing from an input file.<br>
```scanio analyse --scanner bandit --input-file /Users/root/.scanio/output.file```

### Analysing only one repository manually by path
The command uses a path that is pointing to a particular folder for analysing.<br>

If you use a specific **path** argument the command will save results into the same directory:<br>
* ```scanio analyse --scanner <scanner_name> --format sarif /tmp/my_project```
> Result path - ```/tmp/my_project/<scanner_name>.<report_format>```

#### Semgrep
```scanio analyse --scanner semgrep --format sarif /tmp/my_project```

#### Bandit
*TODO*
