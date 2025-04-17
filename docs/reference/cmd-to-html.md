# To HTML Command
The `to-html` command converts sarif, standard sast output format, to a human-friendly html file.

## Table of Contents

- [Syntax](#syntax)
- [Options](#options)
- [Usage Examples](#usage-examples)

## Syntax
```
scanio to-html --input/-i PATH --output/-o PATH [--source/-s PATH] [--templates-path/-t PATH]
```

### Options
| Option | Type | Required | Default Value | Description |
|--------|------|----------|---------------|-------------|
| `--input`, `-i` | string | Yes | `none` | Path to input file, sarif report |
| `--output`, `-o` | string | Yes | `none` | Path to output file, html report
| `--source`, `-s` | string | No | `none` | Path to source code folder |
| `--templates-path`, `-t` | string | No | `none` | Path to templates folder |

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

**Use custom template path**  
Use a custom path to a template file, in case it is located in non standard location or you would like to use customized verion of html template. The target folder should contain only a template with filename `report.html`.
```bash
scanio to-html -i /path/to/project/results.sarif -o /path/to/project/results.html -t /path/to/templates/tohtml
```