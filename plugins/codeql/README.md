# CodeQL Plugin

### Prerequisites
Follow official documentation to install codeql-cli, queries repos and/or qlpacks: https://docs.github.com/en/code-security/codeql-cli/using-the-codeql-cli/getting-started-with-the-codeql-cli.

### Usage Example
list+fetch+scan flow:
```bash
scanio list --vcs github --vcs-url github.com --namespace scan-io-git --output /tmp/scanio-projects.json

scanio fetch --vcs github -i /tmp/scanio-projects.json --auth-type http

# Currently env var 'SCANIO_CODEQL_LANGUAGE' is the only way to say codeql, what is the languages of a project.
export SCANIO_CODEQL_LANGUAGE=go
scanio analyse --scanner codeql -c codeql/go-queries -f sarifv2.1.0 -i /tmp/scanio-projects.json
# results will be saved in dedicated folder for the project: '~/.scanio/results/github.com/scan-io-git/scan-io/'
```
just scan:
```bash
# Currently env var 'SCANIO_CODEQL_LANGUAGE' is the only way to say codeql, what is the languages of a project.
export SCANIO_CODEQL_LANGUAGE=javascript
scanio analyse --scanner codeql -c codeql/javascript-queries -f sarifv2.1.0 /path/to/github.com/juice-shop/juice-shop/
# results will be saved in project rooot directory: '/path/to/github.com/juice-shop/juice-shop/'
```
