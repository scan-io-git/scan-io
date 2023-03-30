# Semgrep plugin
The main function of the plugin is to present a top-level interface for a semgrep scanner. 

This page is a short plugin description.<br>
You may find additional information in a [scanio-analyse](../../docs/scanio-analyse.md) articles.

## Commands
* Analysing using semgrep with an input file argument.<br>
```scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif -j 2```
* Analysing using semgrep with a specific path .<br>
```scanio analyse --scanner semgrep --format sarif -j 1 /tmp/my_project```
* Analysing using semgrep with an input file and custom rules.<br>
```scanio analyse --scanner semgrep --config /Users/root/scan-io-semgrep-rules --input-file /Users/root/.scanio/output.file --format sarif -j 2```
* Analysing using semgrep with an input file and additional arguments.<br>
```scanio analyse --scanner semgrep --input-file /Users/root/.scanio/output.file --format sarif -- --verbose --severity INFO```

## Results of the command
If you use an **input file** argument the command will save results into a home directory: ```~/.scanio/results/+<VCSURL>+<Namespace>+<repo_name>/<scanner_name>.<report_format>```.<br><br>

If you use a specific **path** argument the command will save results into the same directory:<br>
* ```scanio analyse --scanner <scanner_name> --format sarif /tmp/my_project```
* Result path - ```/tmp/my_project/<scanner_name>.<report_format>```r>

## Possible errors
### ```Semgrep does not support Linux ARM64```
You may face with this error if you are using Mac with an M chip. 

You juset need to build a docker container with a platform flag. 
```docker build --platform linux/amd64 -t scanio .```
And use a docker command with the same flas:
```
docker run --rm \                              
            -v "/~/develop/:/data" \
            --platform linux/amd64 \
            scanio analyse --scanner semgrep --input-file /data/testSEC.file --format sarif -j 1
```
