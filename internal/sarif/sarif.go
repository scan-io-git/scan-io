package sarif

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

type Report struct {
	*sarif.Report
	logger       hclog.Logger
	sourceFolder string
}

type ToolMetadata struct {
	Name    string
	Version *string
}

func readSarifReport(inputPath string) (*sarif.Report, error) {
	jsonFile, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	var sarifReport sarif.Report
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal([]byte(byteValue), &sarifReport)

	return &sarifReport, nil
}

// remove all results with Suppressions property
func removeSuppressedResults(report *sarif.Report) {
	for _, run := range report.Runs {
		var filteredResults []*sarif.Result

		for _, result := range run.Results {
			if len(result.Suppressions) == 0 {
				filteredResults = append(filteredResults, result)
			}
		}

		run.Results = filteredResults
	}
}

func ReadReport(inputPath string, logger hclog.Logger, sourceFolder string, noSuppressions bool) (*Report, error) {

	sarifReport, err := readSarifReport(inputPath)
	if err != nil {
		return nil, err
	}

	if noSuppressions {
		removeSuppressedResults(sarifReport)
	}

	// make an absolute path of source folder
	expandedSourceFolder, err := files.ExpandPath(sourceFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to expand source folder: %w", err)
	}
	absPath, err := filepath.Abs(expandedSourceFolder)
	if err != nil {
		return nil, err
	}

	return &Report{
		Report:       sarifReport,
		logger:       logger,
		sourceFolder: absPath,
	}, nil
}

// ExtractToolNameAndVersion function extracts tool name and version from a sarif report
func (r Report) ExtractToolNameAndVersion() (*ToolMetadata, error) {
	toolName := r.Runs[0].Tool.Driver.Name
	toolVersion := r.Runs[0].Tool.Driver.SemanticVersion
	return &ToolMetadata{
		Name:    toolName,
		Version: toolVersion,
	}, nil
}

// CollectSeverityInfo returns counts per severity bucket (critical/high/medium/low/info/unknown)
// and a total. Suppressed results are excluded — only active findings are counted.
// It reads Properties["Severity"] set by EnrichResultsLevelProperty and
// Properties["Suppressed"] set by EnrichResultsSuppressionProperty.
func (r Report) CollectSeverityInfo() map[string]int {
	counts := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
		"info":     0,
		"unknown":  0,
		"total":    0,
	}

	for _, run := range r.Runs {
		for _, result := range run.Results {
			if result.Properties["Suppressed"] == "true" {
				continue
			}
			sev, _ := result.Properties["Severity"].(string)
			if sev == "" {
				sev = "unknown"
			}
			if _, ok := counts[sev]; !ok {
				sev = "unknown"
			}
			if sev != "total" {
				counts[sev]++
			}
			counts["total"]++
		}
	}

	return counts
}

// EnrichResultsSuppressionProperty reads each result's Suppressions field and
// sets Properties["Suppressed"] = "true" for any result that has at least one
// suppression entry (regardless of status), plus SuppressionStatus /
// SuppressionKind / SuppressionJustification for the template.
func (r *Report) EnrichResultsSuppressionProperty() {
	for _, run := range r.Runs {
		for _, result := range run.Results {
			if len(result.Suppressions) == 0 {
				continue
			}
			sup := result.Suppressions[0]
			status := ""
			if sup.Status != nil {
				status = *sup.Status
			}
			result.Properties["Suppressed"] = "true"
			result.Properties["SuppressionStatus"] = status
			result.Properties["SuppressionKind"] = sup.Kind
			if sup.Justification != nil {
				result.Properties["SuppressionJustification"] = *sup.Justification
			}
		}
	}
}

// CollectSuppressionInfo returns counts for suppressed, active, and total findings.
// Must be called after EnrichResultsSuppressionProperty.
func (r Report) CollectSuppressionInfo() map[string]int {
	suppressed := 0
	total := 0
	for _, run := range r.Runs {
		for _, result := range run.Results {
			total++
			if s, _ := result.Properties["Suppressed"].(string); s == "true" {
				suppressed++
			}
		}
	}
	return map[string]int{
		"suppressed": suppressed,
		"active":     total - suppressed,
		"total":      total,
	}
}

var (
	reSentenceBoundary = regexp.MustCompile(`\.\s+[A-Z]`)
	reDashUnderscore   = regexp.MustCompile(`[-_]+`)
)

// humanizeRuleID returns the last dot-segment of ruleID with dashes and underscores
// replaced by spaces and the first letter uppercased.
// "python.lang.security.audit.dangerous-system-call" → "Dangerous system call"
func humanizeRuleID(ruleID string) string {
	if ruleID == "" {
		return ""
	}
	parts := strings.Split(ruleID, ".")
	last := parts[len(parts)-1]
	cleaned := strings.TrimSpace(reDashUnderscore.ReplaceAllString(last, " "))
	if cleaned == "" {
		return ""
	}
	runes := []rune(cleaned)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// firstSentence returns text up to and including the first period that is followed
// by whitespace and an uppercase letter. Returns "" when no boundary is found.
func firstSentence(text string) string {
	loc := reSentenceBoundary.FindStringIndex(text)
	if loc == nil {
		return ""
	}
	return strings.TrimSpace(text[:loc[0]+1])
}

// firstLine returns the first line of text.
func firstLine(text string) string {
	return strings.SplitN(text, "\n", 2)[0]
}

// cap120 truncates s to at most 120 Unicode code points.
func cap120(s string) string {
	if utf8.RuneCountInString(s) <= 120 {
		return s
	}
	return string([]rune(s)[:120])
}

// resolveFindingTitle walks a candidate chain to produce a human-readable title.
// At each non-empty candidate: if it contains the rule ID, short-circuit to the
// humanized rule ID. Otherwise use the candidate. Falls back to humanized rule ID
// (or "Finding") when all candidates are empty.
func resolveFindingTitle(rule *sarif.ReportingDescriptor, result *sarif.Result) string {
	ruleID := strings.TrimSpace(rule.ID)

	// Build candidate list — skip nil pointer fields safely.
	var shortDesc, name, msg string
	if rule.ShortDescription != nil && rule.ShortDescription.Text != nil {
		shortDesc = strings.TrimSpace(*rule.ShortDescription.Text)
	}
	if rule.Name != nil {
		name = strings.TrimSpace(*rule.Name)
	}
	if result.Message.Text != nil {
		msg = strings.TrimSpace(*result.Message.Text)
	}

	humanized := humanizeRuleID(ruleID)
	fallback := humanized
	if fallback == "" {
		fallback = "Finding"
	}

	for _, c := range []string{shortDesc, name, firstSentence(msg), firstLine(msg)} {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if ruleID != "" && strings.Contains(c, ruleID) {
			return cap120(fallback)
		}
		return cap120(c)
	}
	return cap120(fallback)
}

// EnrichResultsTitleProperty function enriches sarif results properties with title and description values
func (r Report) EnrichResultsTitleProperty() {
	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range r.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range r.Runs[0].Results {
		if rule, ok := rulesMap[*result.RuleID]; ok {
			if result.Properties == nil {
				result.Properties = make(map[string]interface{})
			}
			result.Properties["Title"] = resolveFindingTitle(rule, result)
			if result.Message.Text != nil && *result.Message.Text != "" {
				result.Properties["Description"] = *result.Message.Text
			} else if rule.FullDescription != nil && rule.FullDescription.Text != nil {
				result.Properties["Description"] = *rule.FullDescription.Text
			}
		}
	}
}

// EnrichResultsLocationProperty function enriches sarif location properties with source code and URI values
func (r Report) EnrichResultsLocationProperty(location *sarif.Location) error {
	artifactLocation := location.PhysicalLocation.ArtifactLocation
	if artifactLocation.Properties == nil {
		artifactLocation.Properties = make(map[string]interface{})
	}

	// set artifactLocation.Properties["URI"] to be *artifactLocation.URI if it's a relative path,
	// otherwise trim prefix of r.sourceFolder from *artifactLocation.URI
	if !filepath.IsAbs(*artifactLocation.URI) {
		artifactLocation.Properties["URI"] = *artifactLocation.URI
	} else {
		artifactLocation.Properties["URI"] = (*artifactLocation.URI)[len(r.sourceFolder):]
		// remove slash if string start with slash
		if len(artifactLocation.Properties["URI"].(string)) > 0 && artifactLocation.Properties["URI"].(string)[0] == '/' {
			artifactLocation.Properties["URI"] = artifactLocation.Properties["URI"].(string)[1:]
		}
	}

	if location.PhysicalLocation.Region.Properties == nil {
		location.PhysicalLocation.Region.Properties = make(map[string]interface{})
	}
	if location.PhysicalLocation.Region.StartColumn != nil {
		location.PhysicalLocation.Region.Properties["StartColumn"] = *location.PhysicalLocation.Region.StartColumn - 1
	} else {
		location.PhysicalLocation.Region.Properties["StartColumn"] = 0
	}
	if location.PhysicalLocation.Region.EndColumn != nil {
		location.PhysicalLocation.Region.Properties["EndColumn"] = *location.PhysicalLocation.Region.EndColumn - 1
	} else {
		location.PhysicalLocation.Region.Properties["EndColumn"] = 0
	}
	if location.PhysicalLocation.Region.StartLine != nil {
		location.PhysicalLocation.Region.Properties["StartLine"] = *location.PhysicalLocation.Region.StartLine
	} else {
		location.PhysicalLocation.Region.Properties["StartLine"] = 0
	}
	if location.PhysicalLocation.Region.EndLine != nil {
		location.PhysicalLocation.Region.Properties["EndLine"] = *location.PhysicalLocation.Region.EndLine
	} else {
		location.PhysicalLocation.Region.Properties["EndLine"] = location.PhysicalLocation.Region.Properties["StartLine"]
	}

	// Prefer the SARIF-provided snippet text if present — it already has the full multi-line range.
	region := location.PhysicalLocation.Region
	if region != nil && region.Snippet != nil && region.Snippet.Text != nil && *region.Snippet.Text != "" {
		full := *region.Snippet.Text
		artifactLocation.Properties["Code"] = full
		artifactLocation.Properties["CodeLines"] = strings.Split(full, "\n")
		return nil
	}

	// Otherwise read the lines from disk. Requires the source folder to be set.
	if r.sourceFolder == "" {
		return fmt.Errorf("source folder is not set")
	}
	lines, err := r.readLinesFromFile(location.PhysicalLocation)
	if err != nil {
		return err
	}
	artifactLocation.Properties["Code"] = strings.Join(lines, "\n")
	artifactLocation.Properties["CodeLines"] = lines

	return nil
}

// readLinesFromFile reads the [StartLine, EndLine] range from the file referenced by loc.
// Falls back to just StartLine if EndLine is unset.
func (r Report) readLinesFromFile(loc *sarif.PhysicalLocation) ([]string, error) {
	if r.sourceFolder == "" {
		return nil, fmt.Errorf("source folder is not set")
	}
	if loc.Region == nil || loc.Region.StartLine == nil {
		return nil, fmt.Errorf("region or StartLine is nil")
	}

	filePath := *loc.ArtifactLocation.URI
	if !filepath.IsAbs(filePath) {
		fixedFilePath, err := files.ExpandPath(filepath.Join(r.sourceFolder, *loc.ArtifactLocation.URI))
		if err != nil {
			return nil, fmt.Errorf("failed to construct a file path: %w", err)
		}
		filePath = fixedFilePath
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	startLine := *loc.Region.StartLine
	endLine := startLine
	if loc.Region.EndLine != nil && *loc.Region.EndLine >= startLine {
		endLine = *loc.Region.EndLine
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var out []string
	current := 0
	for scanner.Scan() {
		current++
		if current >= startLine && current <= endLine {
			out = append(out, scanner.Text())
		}
		if current >= endLine {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("lines %d-%d not found in file", startLine, endLine)
	}
	return out, nil
}

// EnrichResultsCodeFlowProperty function enriches code flow location properties with source code and URI values
func (r Report) EnrichResultsCodeFlowProperty(locationWebURLCallback func(artifactLocation *sarif.Location) string) {

	for _, result := range r.Runs[0].Results {
		if len(result.CodeFlows) == 0 && len(result.Locations) > 0 {
			//add new code flow
			codeFlow := sarif.NewCodeFlow()
			for _, location := range result.Locations {
				threadFlow := sarif.NewThreadFlow()
				threadFlow.Locations = append(threadFlow.Locations, &sarif.ThreadFlowLocation{
					Location: location,
				})
				codeFlow.ThreadFlows = append(codeFlow.ThreadFlows, threadFlow)
			}
			result.CodeFlows = append(result.CodeFlows, codeFlow)
		}

		for _, codeflow := range result.CodeFlows {
			for _, threadflow := range codeflow.ThreadFlows {
				for _, location := range threadflow.Locations {
					err := r.EnrichResultsLocationProperty(location.Location)
					if err != nil {
						r.logger.Debug("can't read source file", "err", err)
						continue
					}

					if location.Location.Properties == nil {
						location.Location.Properties = make(map[string]interface{})
					}
					location.Location.Properties["WebURL"] = locationWebURLCallback(location.Location)
				}
			}
		}
	}
}

// normalizeSeverityString maps a case-insensitive severity string to canonical (level, severity) pair.
// Returns ok=false when the string is not recognized, so callers can try the next source.
func normalizeSeverityString(raw string) (level, severity string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "critical":
		return "error", "critical", true
	case "high", "error":
		return "error", "high", true
	case "medium", "warning":
		return "warning", "medium", true
	case "low", "note":
		return "note", "low", true
	case "info", "none":
		return "none", "info", true
	default:
		return "", "", false
	}
}

// cvssToBuckets converts a CVSS-style numeric score (0.0-10.0) to (level, severity).
// Thresholds follow the GitHub Code Scanning SARIF spec.
func cvssToBuckets(score float64) (level, severity string) {
	switch {
	case score >= 9.0:
		return "error", "critical"
	case score >= 7.0:
		return "error", "high"
	case score >= 4.0:
		return "warning", "medium"
	case score > 0.0:
		return "note", "low"
	default:
		return "none", "info"
	}
}

// EnrichResultsLevelProperty sets Properties["Level"] (canonical SARIF: error/warning/note/none/unknown)
// and Properties["Severity"] (display bucket: critical/high/medium/low/info/unknown) on every result.
//
// Source precedence (descending granularity):
//  1. rule.Properties["security-severity"] (CVSS numeric 0.0-10.0)
//  2. rule.Properties["problem.severity"] (Semgrep/CodeQL string, may carry critical/info)
//  3. result.Level (SARIF per-finding, only 4-value resolution)
//  4. rule.DefaultConfiguration.Level
//  5. fallback: unknown/unknown
//
// Both properties are idempotent: if both are already set the result is skipped.
func (r Report) EnrichResultsLevelProperty() {
	for _, run := range r.Runs {
		rulesMap := map[string]*sarif.ReportingDescriptor{}
		if run.Tool.Driver != nil {
			for _, rule := range run.Tool.Driver.Rules {
				if rule == nil {
					continue
				}
				rulesMap[rule.ID] = rule
			}
		}

		for _, result := range run.Results {
			if result == nil {
				continue
			}

			if result.Properties == nil {
				result.Properties = make(map[string]interface{})
			}

			// Idempotence: skip if both already set.
			_, hasLevel := result.Properties["Level"]
			_, hasSeverity := result.Properties["Severity"]
			if hasLevel && hasSeverity {
				continue
			}

			var ruleDescriptor *sarif.ReportingDescriptor
			if result.RuleID != nil {
				if rule, ok := rulesMap[*result.RuleID]; ok {
					ruleDescriptor = rule
				}
			}

			level, severity := resolveResultSeverity(result, ruleDescriptor)
			result.Properties["Level"] = level
			result.Properties["Severity"] = severity
		}
	}
}

// resolveResultSeverity walks the source precedence chain and returns (level, severity).
func resolveResultSeverity(result *sarif.Result, rule *sarif.ReportingDescriptor) (level, severity string) {
	// 1. security-severity (CVSS numeric) on the rule.
	if rule != nil && rule.Properties != nil {
		if raw, ok := rule.Properties["security-severity"]; ok && raw != nil {
			if score, ok := toFloat64(raw); ok {
				return cvssToBuckets(score)
			}
		}
	}

	// 2. problem.severity string on the rule.
	if rule != nil && rule.Properties != nil {
		if raw, ok := rule.Properties["problem.severity"]; ok {
			if str, ok := raw.(string); ok {
				if lvl, sev, ok := normalizeSeverityString(str); ok {
					return lvl, sev
				}
			}
		}
	}

	// 3. result.Level (SARIF per-finding).
	if result.Level != nil {
		if lvl, sev, ok := normalizeSeverityString(*result.Level); ok {
			return lvl, sev
		}
	}

	// 4. rule.DefaultConfiguration.Level.
	if rule != nil && rule.DefaultConfiguration != nil {
		if lvl, sev, ok := normalizeSeverityString(rule.DefaultConfiguration.Level); ok {
			return lvl, sev
		}
	}

	return "unknown", "unknown"
}

// toFloat64 coerces interface{} values that may be float64, json.Number, or numeric string to float64.
func toFloat64(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	}
	return 0, false
}

// EnrichResultsLocationURIProperty sets the URI property on the first location of every result
// and calls the provided callbacks to populate WebURL and (optionally) PRWebURL.
// prDiffURLCallback may be nil, in which case PRWebURL is not set.
func (r Report) EnrichResultsLocationURIProperty(
	locationWebURLCallback func(*sarif.Location) string,
	prDiffURLCallback func(*sarif.Location) string,
) {
	for _, result := range r.Runs[0].Results {
		if len(result.Locations) == 0 {
			continue
		}
		location := result.Locations[0]
		artifactLocation := location.PhysicalLocation.ArtifactLocation
		if artifactLocation.URI == nil {
			continue
		}

		if !filepath.IsAbs(*artifactLocation.URI) {
			artifactLocation.Properties["URI"] = *artifactLocation.URI
		} else {
			artifactLocation.Properties["URI"] = (*artifactLocation.URI)[len(r.sourceFolder):]
			if len(artifactLocation.Properties["URI"].(string)) > 0 && artifactLocation.Properties["URI"].(string)[0] == '/' {
				artifactLocation.Properties["URI"] = artifactLocation.Properties["URI"].(string)[1:]
			}
		}

		if location.Properties == nil {
			location.Properties = make(map[string]interface{})
		}
		location.Properties["WebURL"] = locationWebURLCallback(location)
		if prDiffURLCallback != nil {
			location.Properties["PRWebURL"] = prDiffURLCallback(location)
		}
	}
}

// SortResultsBySeverity sorts results by Severity bucket: critical < high < medium < low < info < unknown.
func (r Report) SortResultsBySeverity() {
	severityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
		"info":     4,
		"unknown":  5,
	}

	for _, run := range r.Runs {
		sort.Slice(run.Results, func(i, j int) bool {
			si, _ := run.Results[i].Properties["Severity"].(string)
			sj, _ := run.Results[j].Properties["Severity"].(string)
			oi, ok := severityOrder[si]
			if !ok {
				oi = 5
			}
			oj, ok := severityOrder[sj]
			if !ok {
				oj = 5
			}
			return oi < oj
		})
	}
}

// remove codeflow duplicates
// each codeflow may have multiple threatflows. These threatflows may be equal for different codeflows.
// This function removes duplicates from codeflows
// if the codeflow is empty, it is removed
func (r Report) RemoveDataflowDuplicates() {
	for _, run := range r.Runs {
		for _, result := range run.Results {
			uniqueThreadFlowsFingerprints := map[string]bool{}
			for _, codeFlow := range result.CodeFlows {
				uniqueThreadFlows := []*sarif.ThreadFlow{}
				for _, threadFlow := range codeFlow.ThreadFlows {
					fingerprint := calculateThreadFlowFingerprint(threadFlow)
					if _, ok := uniqueThreadFlowsFingerprints[fingerprint]; !ok {
						uniqueThreadFlowsFingerprints[fingerprint] = true
						uniqueThreadFlows = append(uniqueThreadFlows, threadFlow)
					}
				}
				codeFlow.ThreadFlows = uniqueThreadFlows
			}

			// remove empty codeflows
			nonEmptyCodeFlows := []*sarif.CodeFlow{}
			for _, codeFlow := range result.CodeFlows {
				if len(codeFlow.ThreadFlows) > 0 {
					nonEmptyCodeFlows = append(nonEmptyCodeFlows, codeFlow)
				}
			}
			result.CodeFlows = nonEmptyCodeFlows
		}
	}
}

// function that calculates a fingerprint for threadflow
func calculateThreadFlowFingerprint(threadFlow *sarif.ThreadFlow) string {
	var fingerprint string
	for _, location := range threadFlow.Locations {
		fingerprint += fmt.Sprintf("|%s:%d:%d:%d:%d;",
			location.Location.PhysicalLocation.ArtifactLocation.Properties["URI"].(string),
			location.Location.PhysicalLocation.Region.Properties["StartLine"].(int),
			location.Location.PhysicalLocation.Region.Properties["StartColumn"].(int),
			location.Location.PhysicalLocation.Region.Properties["EndLine"].(int),
			location.Location.PhysicalLocation.Region.Properties["EndColumn"].(int),
		)
	}
	return calculateMD5Hash(fingerprint)
}

// function that calculates md5 hash for a given text
func calculateMD5Hash(text string) string {
	hash := md5.New()
	io.WriteString(hash, text)
	return hex.EncodeToString(hash.Sum(nil))
}

// EnrichResultsCategoryProperty writes Properties["Category"] (display label) and
// Properties["CategorySlug"] (machine value) for each result. Defaults to CategoryOther
// when no CWE tag or rule-ID keyword matches.
func (r Report) EnrichResultsCategoryProperty() {
	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range r.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range r.Runs[0].Results {
		rule := rulesMap[*result.RuleID]
		cat, ok := resolveCategory(*result.RuleID, rule)
		if !ok {
			cat = CategoryOther
		}
		if result.Properties == nil {
			result.Properties = make(map[string]any)
		}
		result.Properties["Category"] = categoryLabels[cat]
		result.Properties["CategorySlug"] = string(cat)
	}
}

// EnrichResultsConfidenceProperty writes Properties["Confidence"] (e.g. "High (85%)")
// for each result. The key is omitted when no confidence signal is found.
func (r Report) EnrichResultsConfidenceProperty() {
	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range r.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range r.Runs[0].Results {
		rule := rulesMap[*result.RuleID]
		conf, ok := resolveConfidence(result, rule)
		if !ok {
			continue
		}
		if result.Properties == nil {
			result.Properties = make(map[string]any)
		}
		result.Properties["Confidence"] = formatConfidence(conf)
	}
}

// EnrichResultsMetadataProperty writes RuleFull, RuleShort, Scanner, CodeSectionLabel,
// References, and Fix onto each result's Properties map. Fields are omitted when empty.
// Must run after EnrichResultsCodeFlowProperty and RemoveDataflowDuplicates so that
// CodeSectionLabel reflects the final thread-flow count.
func (r Report) EnrichResultsMetadataProperty() {
	scanner := r.Runs[0].Tool.Driver.Name

	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range r.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range r.Runs[0].Results {
		if result.Properties == nil {
			result.Properties = make(map[string]any)
		}

		ruleID := ""
		if result.RuleID != nil {
			ruleID = *result.RuleID
		}

		// Rule ID chips.
		if ruleID != "" {
			result.Properties["RuleFull"] = ruleID
			short := ruleID
			if i := strings.LastIndex(ruleID, "."); i >= 0 {
				short = ruleID[i+1:]
			}
			result.Properties["RuleShort"] = short
		}

		// Scanner name.
		if scanner != "" {
			result.Properties["Scanner"] = scanner
		}

		// Code section label — "Data flow" when a real multi-step thread flow exists.
		label := "Affected code"
		if len(result.CodeFlows) > 0 && len(result.CodeFlows[0].ThreadFlows) > 0 {
			if len(result.CodeFlows[0].ThreadFlows[0].Locations) > 1 {
				label = "Data flow"
			}
		}
		result.Properties["CodeSectionLabel"] = label

		// References and Fix.
		rule := rulesMap[ruleID]
		if refs := extractReferences(result, rule, 3); len(refs) > 0 {
			result.Properties["References"] = refs
		}
		if parts := splitFixParts(extractFix(result, rule)); len(parts) > 0 {
			result.Properties["FixParts"] = parts
		}
	}
}
