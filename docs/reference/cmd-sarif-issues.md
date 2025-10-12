# SARIF Issues Command
The `sarif-issues` command creates GitHub issues from SARIF findings with configurable severity levels. It implements intelligent issue correlation to avoid duplicates and automatically closes issues that are no longer present in recent scans.

This command is designed and recommended for CI/CD integration and automated security issue management, enabling teams to track and manage security findings directly in their GitHub repositories.

## Table of Contents

- [Key Features](#key-features)
- [Syntax](#syntax)
- [Options](#options)
- [Severity Level Configuration](#severity-level-configuration)
- [Core Validation](#core-validation)
- [Dry Run Mode](#dry-run-mode)
- [GitHub Authentication Setup](#github-authentication-setup)
- [Usage Examples](#usage-examples)
- [Command Output Format](#command-output-format)
- [Issue Correlation Logic](#issue-correlation-logic)
- [Issue Format](#issue-format)

## Key Features

| Feature                                   | Description                                              |
|-------------------------------------------|----------------------------------------------------------|
| Create issues from configurable severity levels | Automatically creates GitHub issues for SARIF findings with specified severity levels (default: "error") |
| Correlate with existing issues           | Matches new findings against open issues to prevent duplicates |
| Auto-close resolved issues               | Closes open issues that are no longer present in current scan results |
| Add metadata and permalinks              | Enriches issues with file links, severity, scanner info, and code snippets |

## Syntax
```bash
scanio sarif-issues --sarif PATH [--namespace NAMESPACE] [--repository REPO] [--source-folder PATH] [--ref REF] [--labels label[,label...]] [--assignees user[,user...]] [--levels level[,level...]] [--dry-run]
```

## Options

| Option              | Type     | Required    | Default Value                    | Description                                                                 |
|---------------------|----------|-------------|----------------------------------|-----------------------------------------------------------------------------|
| `--sarif`           | string   | Yes         | `none`                           | Path to SARIF report file containing security findings.                    |
| `--namespace`       | string   | Conditional | `$GITHUB_REPOSITORY_OWNER`       | GitHub organization or user name. Required if environment variable not set. |
| `--repository`      | string   | Conditional | `${GITHUB_REPOSITORY#*/}`        | Repository name. Required if environment variable not set.                  |
| `--source-folder`   | string   | No          | `.`                              | Path to source code folder for improved file path resolution and snippets. |
| `--ref`             | string   | No          | `$GITHUB_SHA`                    | Git ref (branch or commit SHA) for building permalinks to vulnerable code. |
| `--labels`          | strings  | No          | `none`                           | Labels to assign to created GitHub issues (comma-separated or repeat flag). |
| `--assignees`       | strings  | No          | `none`                           | GitHub usernames to assign to created issues (comma-separated or repeat flag). |
| `--levels`          | strings  | No          | `["error"]`                      | SARIF severity levels to process. Accepts SARIF levels (error, warning, note, none) or display levels (High, Medium, Low, Info). Cannot mix formats. Case-insensitive. |
| `--help`, `-h`      | flag     | No          | `false`                          | Displays help for the `sarif-issues` command.                              |

**Environment Variable Fallbacks**<br>
The command automatically uses GitHub Actions environment variables when flags are not provided:
- `GITHUB_REPOSITORY_OWNER` → `--namespace`
- `GITHUB_REPOSITORY` → `--repository` (extracts repo name after `/`)
- `GITHUB_SHA` → `--ref`

This enables seamless integration with GitHub Actions workflows without explicit configuration.

## Severity Level Configuration

The `--levels` flag allows you to specify which SARIF severity levels should trigger issue creation. This provides flexibility in managing different types of security findings based on your team's priorities.

### Supported Level Formats

**SARIF Levels** (native SARIF format):
- `error` - High severity findings (default)
- `warning` - Medium severity findings  
- `note` - Low severity findings
- `none` - Informational findings

**Display Levels** (human-readable format):
- `High` - Maps to SARIF `error`
- `Medium` - Maps to SARIF `warning`
- `Low` - Maps to SARIF `note` 
- `Info` - Maps to SARIF `none`

### Usage Rules

- **Case-insensitive**: All level comparisons are case-insensitive
- **Format consistency**: Cannot mix SARIF and display levels in the same command
- **Multiple values**: Use comma-separated values or repeat the flag
- **Default behavior**: When `--levels` is not specified, only `error` level findings create issues

### Examples

```bash
# Default behavior (error level only)
scanio sarif-issues --sarif report.sarif

# Multiple SARIF levels
scanio sarif-issues --sarif report.sarif --levels error,warning

# Multiple display levels  
scanio sarif-issues --sarif report.sarif --levels High,Medium

# All severity levels using SARIF format
scanio sarif-issues --sarif report.sarif --levels error,warning,note,none

# Invalid mixing (will error)
scanio sarif-issues --sarif report.sarif --levels error,High
```

## Core Validation
The `sarif-issues` command includes several validation layers to ensure robust execution:
- **Required Parameters**: Validates that `--sarif`, `--namespace`, and `--repository` are provided either via flags or environment variables.
- **SARIF File Validation**: Ensures the SARIF file exists and can be parsed successfully.
- **GitHub Authentication**: Requires valid GitHub credentials configured through the GitHub plugin.
- **Severity Level Validation**: Validates and normalizes severity levels, preventing mixing of SARIF and display level formats.
- **Configurable Severity Filtering**: Processes SARIF results based on specified severity levels (default: "error" only).

## Dry Run Mode

The `--dry-run` flag allows you to preview what the command would do without making actual GitHub API calls. This is particularly useful for:

- **Testing and Validation**: Verify the command behavior before running in production
- **Understanding Impact**: See exactly what issues would be created or closed
- **Debugging**: Troubleshoot issue correlation logic and SARIF processing
- **CI/CD Integration**: Validate SARIF files and command configuration

### Dry Run Output Format

When using `--dry-run`, the command provides detailed preview information:

**For issues to be created:**
```
[DRY RUN] Would create issue:
  Title: [Semgrep][High][sql-injection] at app.py:11-29
  File: apps/demo/main.py
  Lines: 11-29
  Severity: High
  Scanner: Semgrep
  Rule ID: sql-injection
```

**For issues to be closed:**
```
[DRY RUN] Would close issue #42:
  File: apps/demo/old-file.py
  Lines: 5-10
  Rule ID: deprecated-rule
  Reason: Not found in current scan
```

**Final summary:**
```
[DRY RUN] Would create 3 issue(s); would close 1 resolved issue(s)
```

### Usage Example
```bash
scanio sarif-issues --sarif results/semgrep.sarif --dry-run
```

## GitHub Authentication Setup

The `sarif-issues` command requires GitHub authentication to create and manage issues. Configure authentication using one of the following methods:

### Environment Variables (Recommended for CI/CD)
```bash
export SCANIO_GITHUB_TOKEN="your-github-token"
export SCANIO_GITHUB_USERNAME="your-github-username"  # Optional for HTTP auth
```

### Configuration File
Add to your `config.yml`:
```yaml
github_plugin:
  token: "your-github-token"
  username: "your-github-username"  # Optional for HTTP auth
```

### Required Token Permissions
The GitHub token must have the following scopes:
- **`repo`** - Required for creating, updating, and listing issues
- **`read:org`** - Required for organizational repositories (optional for personal repos)

For detailed GitHub plugin configuration, refer to [GitHub Plugin Documentation](plugin-github.md#configuration-references).

## Usage Examples

> **Recommendation:** Run the command from your repository root and pass `--source-folder` as repo-relative paths (for example `--source-folder apps/demo`). The flag defaults to `.` when omitted; if git metadata cannot be detected, permalinks and snippet hashing may be incomplete.

### Basic Usage in GitHub Actions
Create issues from SARIF report using environment variables:
```bash
scanio sarif-issues --sarif results/semgrep.sarif
```

### Manual Usage with Explicit Parameters
Create issues with custom namespace and repository:
```bash
scanio sarif-issues --namespace scan-io-git --repository scan-io --sarif /path/to/report.sarif
```

### Enhanced Issue Creation
Create issues with source code snippets, labels, and assignees:
```bash
scanio sarif-issues --sarif results/semgrep.sarif --source-folder . --labels bug,security --assignees alice,bob
```

### Configurable Severity Levels
Create issues for multiple severity levels using SARIF levels:
```bash
scanio sarif-issues --sarif results/semgrep.sarif --levels error,warning
```

Create issues for multiple severity levels using display levels:
```bash
scanio sarif-issues --sarif results/semgrep.sarif --levels High,Medium
```

### With Custom Git Reference
Create issues with specific commit reference for permalinks:
```bash
scanio sarif-issues --sarif results/semgrep.sarif --source-folder . --ref feature-branch
```

### Dry Run Mode
Preview what issues would be created/closed without making actual GitHub API calls:
```bash
scanio sarif-issues --sarif results/semgrep.sarif --dry-run
```

This is useful for:
- Testing and validation before running in production
- Understanding what the command would do without making changes
- Debugging issue correlation logic
- Verifying SARIF file processing

## Command Output Format

### Normal Mode Output
```
Created 3 issue(s); closed 1 resolved issue(s)
```

### Dry Run Mode Output
When using `--dry-run`, the command shows detailed preview information:

**For issues to be created:**
```
[DRY RUN] Would create issue:
  Title: [Semgrep][High][sql-injection] at app.py:11-29
  File: apps/demo/main.py
  Lines: 11-29
  Severity: High
  Scanner: Semgrep
  Rule ID: sql-injection
```

**For issues to be closed:**
```
[DRY RUN] Would close issue #42:
  File: apps/demo/old-file.py
  Lines: 5-10
  Rule ID: deprecated-rule
  Reason: Not found in current scan
```

**Final summary:**
```
[DRY RUN] Would create 3 issue(s); would close 1 resolved issue(s)
```

### Logging Information
The command provides some logging information including:
- Number of open issues fetched from the repository
- Issue correlation results (matched/unmatched)
- Created and closed issue counts
- Error details for failed operations

## Issue Correlation Logic

The command implements intelligent issue correlation to manage the lifecycle of security findings:

### New Issue Creation
- **Configurable Severity Levels**: Creates issues for SARIF findings with specified severity levels (default: "error" only)
- **Duplicate Prevention**: Uses hierarchical correlation to match new findings against existing open issues
- **Unmatched Findings**: Creates GitHub issues only for findings that don't match existing open issues through any correlation stage

### Automatic Issue Closure
- **Resolved Findings**: Automatically closes open issues that don't correlate with current scan results
- **Comment Before Closure**: Adds comment `Recent scan didn't see the issue; closing this as resolved.`.
- **Managed Issues Only**: Only closes issues containing the scanio-managed annotation to avoid affecting manually created issues

### Correlation Criteria
The correlation logic uses a **4-stage hierarchical matching system** that processes stages in order, with earlier stages being more precise. Once an issue is matched in any stage, it's excluded from subsequent stages.

**Required for all stages**: Scanner name and Rule ID must match exactly.

**Stage 1 (Most Precise)**: Scanner + RuleID + Filename + StartLine + EndLine + SnippetHash
- All fields must match exactly
- Used when both issues have snippet hashes available

**Stage 2**: Scanner + RuleID + Filename + SnippetHash  
- Matches based on code content fingerprint
- Used when snippet hashes are available but line numbers differ

**Stage 3**: Scanner + RuleID + Filename + StartLine + EndLine
- Matches based on exact line range
- Used when snippet hashes are not available

**Stage 4 (Least Precise)**: Scanner + RuleID + Filename + StartLine
- Matches based on file and starting line only
- Fallback when end line information is missing

### Issue Filtering for Correlation
Only specific types of open issues are considered for correlation:
- **Well-structured issues**: Must have Scanner, RuleID, and FilePath metadata
- **Scanio-managed issues**: Must contain the scanio-managed annotation
- **Malformed issues are skipped**: Issues without proper metadata are ignored to prevent accidental closure of manually created issues

### Subfolder Scoping
The command supports independent issue management for different subfolders in monorepo workflows:

- **Scoped Correlation**: When `--source-folder` points to a subfolder, only open issues whose file paths fall within that subfolder are considered for correlation
- **Independent Management**: Issues from different subfolders are managed independently, preventing cross-subfolder interference
- **Root Scope**: When scanning from repository root (no `--source-folder` or `--source-folder` points to root), all issues are considered

**Example Monorepo Workflow**:
```bash
# Frontend CI job - manages issues in apps/frontend only
scanio sarif-issues --sarif frontend-results.sarif --source-folder apps/frontend

# Backend CI job - manages issues in apps/backend only  
scanio sarif-issues --sarif backend-results.sarif --source-folder apps/backend
```

This enables separate CI jobs for different parts of a monorepo without issues from one subfolder affecting the other.

## Issue Format

### Issue Title Format
```
[<scanner>][<severity>][<rule name or ID>] at <file>:<line>[-<endLine>]
```
**Example**: `[Semgrep OSS][High][Express Missing CSRF Protection] at app.js:42-45`
When a rule provides a human-friendly `name`, Scanio uses it; otherwise the rule ID is shown.

### Issue Body Structure

**Header**
```markdown
## 🐞 <Rule Short Description>
```
Scanio prefers the SARIF rule's short description for the heading; if that is missing it falls back to the rule name, then to the raw rule ID.

**Compact Metadata (Blockquote)**
```markdown
> **Rule ID**: <ruleID>
> **Severity**: High,  **Scanner**: Semgrep OSS
> **File**: app.js, **Lines**: 42-45
```

**Rule Description**
- Includes help text from SARIF rule definitions
- Parses and formats reference links
- Falls back to the rule's full description when markdown help is not available

**GitHub Permalink**
- Direct link to vulnerable code in repository
- Uses commit SHA for permanent links
- Includes line number anchors: `#L42-L45`

**Security References**
- Automatically generates links for CWE identifiers: `[CWE-79](https://cwe.mitre.org/data/definitions/79.html)`
- Creates OWASP Top 10 links when applicable
- Extracts from SARIF rule tags and properties

**Snippet Hash**
```markdown
> **Snippet SHA256**: abc123...
```

**Management Annotation**
```markdown
> [!NOTE]
> This issue was created and will be managed by scanio automation. Don't change body manually for proper processing, unless you know what you do
```

### Example Complete Issue Body
```markdown
## 🐞 javascript.express.security.audit.express-check-csurf-middleware-usage.express-check-csurf-middleware-usage

> **Rule ID**: javascript.express.security.audit.express-check-csurf-middleware-usage.express-check-csurf-middleware-usage
> **Severity**: High,  **Scanner**: Semgrep OSS
> **File**: app.js, **Lines**: 42-45

This Express.js application appears to be missing CSRF protection middleware. CSRF attacks can force authenticated users to perform unintended actions.

https://github.com/scan-io-git/scan-io/blob/abc123def456/app.js#L42-L45

<b>References:</b>
- [CWE-352](https://cwe.mitre.org/data/definitions/352.html)
- [OWASP A01:2021 - Broken Access Control](https://owasp.org/Top10/A01_2021-Broken_Access_Control/)

> **Snippet SHA256**: abc123def456789...

> [!NOTE]
> This issue was created and will be managed by scanio automation. Don't change body manually for proper processing, unless you know what you do
```

This format provides comprehensive information while maintaining machine readability for correlation and automated management.
