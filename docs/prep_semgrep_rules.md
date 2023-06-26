# Rules preparation script 
`scripts/prep_semgrep_rules.sh` is a script that prepares semgrep rules, by fetching remote repositories and copying rules into the `/rules` folder. This could be used for preparing a list of verified rules to put into a docker container. A common scenario is when you prepare a list of rules, which provide good findings, which you want to show to developers.  
After building a docker image, one can use `scanio analyse --scanner semgrep -c /scanio-rules/semgrep /path/to/project` or natively `semgrep -c /scanio-rules/semgrep`.<br><br>

## Examples
**Default mode**<br>
```./prep_semgrep_rules.sh```<br>
The script will download rules from a repo - https://github.com/returntocorp/semgrep-rules.git and branch "release" to a temp directory. From the temp directory rules will be copied that match with names from a rules_semgrep.txt file. The temp directory will be deleted.<br><br>

**Manual rules fetching mode**<br>
```./prep_semgrep_rules.sh -r rules_semgrep.txt https://github.com/returntocorp/semgrep-rules.git release```<br>
The script will download rules from a repo - https://github.com/returntocorp/semgrep-rules.git and branch "release" to a temp directory. From the temp directory rules will be copied that match with names from a rules_semgrep.txt file. The temp directory will be deleted.<br><br>

```./prep_semgrep_rules.sh -r rules_trailofbits.txt https://github.com/trailofbits/semgrep-rules.git main```<br>
The script will download rules from a repo - https://github.com/trailofbits/semgrep-rules.git and branch "main" to a temp directory. From the temp directory rules will be copied that match with names from a rules_trailofbits.txt file. The temp directory will be deleted.<br><br>

**Mergin mode**<br>
The command merges rules from a specified directory to a "semgrep-rules" folder.<br>
```./prep_semgrep_rules.sh -m /scan-io/rules/semgrep/returntocorp```<br>
The script will copy all rules to a "semgrep-rules" folder and delete the "returntocorp" folder.<br><br>

```./prep_semgrep_rules.sh -m /scan-io/rules/semgrep/trailofbits```<br>
The script will copy all rules to a "semgrep-rules" folder and delete the "trailofbits" folder.<br><br>

**Auto mode**<br>
```./prep_semgrep_rules.sh -a```
The script will do all the previous steps:
* Fetch rules from a semgrep repo and branch "release" to a temp directory.
* From the temp directory rules will be copied that match with names from a rules_semgrep.txt file.
* Fetch rules from a trailofbits repo and branch "main" to a temp directory.
* From the temp directory rules will be copied that match with names from a rules_trailofbits.txt file.
* All rules will be merged from a specified directory to a "semgrep-rules" folder.<br><br>

**Diff mode**<br>
TODO<br>
The script has to compare a new revision of the repo with current local rules and create a diff with new or updated rules.<br>
