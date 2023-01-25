# Scanio Analyse Command
The main function is to present a top-level interface for a specified scanner. The function is handling scanners plugins. <br><br>

|Scanners|Semgrep|Bandit|
|----|-----|---|
|Inherited args|Supported|Not supported|
|Custom config for rules|Supported|Not supported|
|Report format arg|Supported|Not supported|
|Local multithread|Supported|Supported|
<br>

## Args of the command
- "scanner" is the plugin name of the scanner used. The default is semgrep.
- "input-file" or "f" is a file in scanio format with a list of repositories to analyse. The list command could prepare this file.
- "format" or "o" is a format for a report with results. 
- "config" or "c" is a path or type of config for a scanner. The value depends on a particular scanner's used formats. The default is auto. 
- "arg" are additional commands for semgrep which will be added to a semgrep call. The format in quotes with commas without spaces, e.g. ```--arg --verbose,--severity,INFO```.
- "threads" or "j" is a number of concurrent goroutines. The default is 1. 
<br><br>

## Using scenarios 
When developing, we aimed at the fact that the program will be used primarily for automation purposes but you still able to use it manually from CLI.<br>

The command is saving results into a home directory ```~/.scanio/results/+<VCSURL>+<Namespace>+<repo_name>/<scanner_name>.<report_format>```.<br><br>

### Analysing from an input file
The command uses an output format of a List command for analysing required repositories.<br><br>

#### **Semgrep**
* Analysing from an input file.<br>
```scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif```
* Analysing from an input file with custom rules.<br>
```scanio analyse --scanner semgrep --config /Users/root/scan-io-semgrep-rules --input-file /Users/root/.scanio/output.file --format sarif```
* Analysing from an input file with additional agruments.<br>
```scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif --args "--verbose,--severity,INFO"```<br><br>

#### **Bandit**
* Analysing from an input file.<br>
```scanio analyse --scanner bandit --input-file /Users/root/.scanio/output.file```

### Analysing only one repository manually by path
The command uses a path that is pointing to a particular repository for analysing.<br><br>

#### **Semgrep**
TODO<br><br>

#### **Bandit**
TODO<br><br>
