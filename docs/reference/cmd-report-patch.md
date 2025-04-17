# Report Patch Command
The `report-patch` command helps to make structured modifications in SARIF reports.  
There are tools, that do not provide enough flexibility to control different data like criticality field, what findings to exclude and so on. The `report-patch` command facilitates adjustments over these fields and findings.

## Table of Contents

- [Syntax](#syntax)
- [Options](#options)
- [Core Validation](#core-validation)
- [Usage Examples](#usage-examples)

## Syntax
```bash
scanio report-patch --input/-i PATH --output/-o PATH [--when-rule STRING] [--when-location-contains STRING_ARRAY] [--when-text-contains STRING_ARRAY] [--when-text-not-contains STRING_ARRAY] [--set-severity SEVERITY] [--delete]
```

### Options

**Supported Selectors**
Selectors allows to apply a modification to a particular subset of rules and findings by applying different conditions.  
Selectors are optional, and can be combined together
| Option | Description | Example |
|--------|-------------|---------|
| `when-rule` | filter by rule name | Useful for adjusting severity for the whole rule |
| `when-location-contains` | filter by finding path | Useful for excluding findings from specific directories (e.g., `/node_modules/`) |
| `when-text-contains` | filter by finding description | Useful for adjusting severity of findings based on context, such as when a potential vulnerability originates from environment variables rather than http request, assuming that it's more trusted source of data |
| `when-text-not-contains` | filter by finding description. Opposite to `when-text-contains` | |

**Supported Modification Actions**
| Option | Description |
|--------|-------------|
| `set-severity` | set findings or rule criticality. Supported values: `high`, `medium`, `low` |
| `delete` | delete findings or rule from a report |

## Usage Examples

**Overwrite severity for specific rule by rule id and results for this rule:**
```bash
scanio report-patch -i original-report.sarif -o patched-report.sarif --when-rule java/CSRFDisabled --set-severity low
``` 

**Set lower criticality for results of specific rule, when description says that source of malicious data is coming from command line arguments:**
```bash
scanio report-patch -i original-report.sarif -o patched-report.sarif --when-rule javascript/SQLInjection --when-text-contains "input from a command line argument" --set-severity low
```

**Delete results found in node_modules subfolder:**
```bash
scanio report-patch -i original-report.sarif -o patched-report.sarif --when-location-contains node_modules/ --delete
```
