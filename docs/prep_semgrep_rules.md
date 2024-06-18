# Rules Preparation Script 
`scripts/prep_semgrep_rules.sh` is a script that prepares semgrep rules, by fetching remote repositories and copying rules into the `/rules` folder. This could be used for preparing a list of verified rules to put into a docker container. A common scenario is when you prepare a list of rules, which provide good findings, which you want to show to developers.  
After building a docker image, one can use `scanio analyse --scanner semgrep -c /scanio/rules/semgrep/default /path/to/project` or natively `semgrep -c /rules/semgrep/default`.<br>

`scripts/prep_semgrep_rules.sh` is a script that prepares Semgrep rules by fetching them from remote repositories and copying the rules into the `/rules` folder. This script is useful for preparing a list of verified rules to include in a Docker container. A common scenario is to prepare a list of rules that provide valuable findings to present to developers.<br>

After building a Docker image, one can use:
* `scanio analyse --scanner semgrep -c /scanio/rules/semgrep/<rule_set_name> /path/to/project`
* Or natively: `semgrep -c /rules/semgrep/<rule_set_name>`<br>

## Examples
**Default Mode**<br>
```./prep_semgrep_rules.sh```<br>
The script will download rules from the repository https://github.com/returntocorp/semgrep-rules.git on the "release" branch to a temporary directory. From the temporary directory, rules matching the names in `default_rules_semgrep.txt` will be copied. The temporary directory will then be deleted.<br><br>

**Manual Rules Fetching Mode**<br>
```./prep_semgrep_rules.sh -r default_rules_semgrep.txt https://github.com/returntocorp/semgrep-rules.git release```<br>
The script will download rules from the repository https://github.com/returntocorp/semgrep-rules.git on the "release" branch to a temporary directory. Rules matching the names in `default_rules_semgrep.txt` will be copied from the temporary directory. The temporary directory will then be deleted.<br><br>

```./prep_semgrep_rules.sh -r default_rules_trailofbits.txt https://github.com/trailofbits/semgrep-rules.git main```<br>
The script will download rules from the repository https://github.com/trailofbits/semgrep-rules.git on the "main" branch to a temporary directory. Rules matching the names in `default_rules_trailofbits.txt` will be copied from the temporary directory. The temporary directory will then be deleted.<br><br>

**Merging Mode**<br>
The command merges rules from a specified directory to a "semgrep-rules" folder.<br>
```./prep_semgrep_rules.sh -m /scan-io/rules/semgrep/returntocorp```<br>
The script will copy all rules to a `semgrep-rules` folder and delete the `returntocorp` folder.<br><br>

```./prep_semgrep_rules.sh -m /scan-io/rules/semgrep/trailofbits```<br>
The script will copy all rules to a `semgrep-rules` folder and delete the `trailofbits` folder.<br><br>

**Auto mode**<br>
```./prep_semgrep_rules.sh -a```<br>

The script will perform the following steps:
* Fetch rules from the Semgrep repository on the "release" branch to a temporary directory.
* Copy rules that match names in default_rules_semgrep.txt from the temporary directory.
* Fetch rules from the Trail of Bits repository on the "main" branch to a temporary directory.
* Copy rules that match names in default_rules_trailofbits.txt from the temporary directory.
Merge all rules from the specified directories into a "semgrep" folder.<br><br>

**Diff Mode**<br>
TODO: The script will compare a new revision of the repository with the current local rules and create a diff with new or updated rules.<br>
