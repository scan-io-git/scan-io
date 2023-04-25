`scripts/prep_semgrep_rules.sh` is a script that prepares semgrep rules, by fetching remote repos and copying rules into `/rules` folder. This could be used for preparing list of verified rules to put into the docker container. Common scenario is when you prepare a list of rules, that you want to show for developers.  
After building a docker image, developers can use `semgrep -c /scanio-rules`.
