# To HTML Command
The `to-html` command converts sarif, standard sast output format, to a human-friendly html file.

## Table of Contents

- [Syntax](#syntax)
- [Options](#options)
- [Usage Examples](#usage-examples)

## Syntax
```
scanio to-html --input/-i PATH --output/-o PATH [--source/-s PATH] [--templates-path/-t PATH] [--no-supressions]
```

### Options
| Option | Type | Required | Default Value | Description |
|--------|------|----------|---------------|-------------|
| `--input`, `-i` | string | Yes | `none` | Path to input file, sarif report |
| `--output`, `-o` | string | Yes | `none` | Path to output file, html report
| `--source`, `-s` | string | No | `none` | Path to source code folder |
| `--templates-path`, `-t` | string | No | `none` | Path to templates folder |
| `--no-supressions` | bool | No | `false` | Enable removing results with suppressions properties |

## Usage Examples
The following examples demonstrate how to use the `to-html` command.

**Basic**  
Convert sarif output to html report, without code snippets.
```bash
scanio to-html -i /path/to/project/results.sarif -o /path/to/project/results.html
```

**With code snippets**  
Convert sarif output to html report with code snippets. Add a source code folder argument, so the tool can extract code snippets for corresponding code flows and locations in a report.
```bash
scanio to-html -i /path/to/project/results.sarif -o /path/to/project/results.html -s /path/to/project
```

**If no template path specified**  
If template path is not specified, the tool will look for templates in home folder: `SCANIO_HOME/templates/tohtml`. `SCANIO_HOME` can be configured in an AppConfig with `scanio.home_folder` key.

**Use custom template path**  
Use a custom path to a template file, in case it is located in non standard location or you would like to use customized verion of html template. The target folder should contain only a template with filename `report.html`.
```bash
scanio to-html -i /path/to/project/results.sarif -o /path/to/project/results.html -t /path/to/templates/tohtml
```

**Ignore Suppressed Findings**
The suppressions property in a SARIF result indicates that the finding was intentionally ignored, either in the source code or through external configuration. 
For example, Semgrep includes rules that were ignored using `// nosemgrep` in the SARIF results and marks them with a [suppressions property](https://docs.oasis-open.org/sarif/sarif/v2.0/csprd02/sarif-v2.0-csprd02.html#_Toc10127852). However, these are still listed as findings, which can be confusing compared to other output formats (e.g., JSON), where such suppressed issues are omitted entirely.

If you want to exclude suppressed results from the HTML report, use the `--no-supressions` flag.
```bash
scanio to-html -i /tmp/juice-shop/semgrep_results.sarif -o /tmp/juice-shop/semgrep_results.html -s /tmp/juice-shop/ -t ./templates/tohtml --no-supressions
```

