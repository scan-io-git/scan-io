package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

const (
	scanStateComplete  = "complete"
	toolName           = "betterleaks"
	toolInformationURI = "https://github.com/betterleaks/betterleaks"
	fingerprintKey     = "betterleaks/v1"
	severityTagPrefix  = "severity:"
)

// Report is the betterleaks v2 JSON envelope. Author identity
// (git.author_*, git.committer_*) is deliberately not read from the attributes,
// so it cannot reach the SARIF output.
type Report struct {
	SchemaVersion string    `json:"schema_version"`
	Findings      []Finding `json:"findings"`
	Scan          Scan      `json:"scan"`
}

// Scan is the summary block at the end of the report.
type Scan struct {
	State              string `json:"state"`
	BetterleaksVersion string `json:"betterleaks_version"`
	ConfigHash         string `json:"config_hash"`
	BytesScanned       int64  `json:"bytes_scanned"`
	NumFindings        int    `json:"num_findings"`
}

// Finding is a single betterleaks finding.
type Finding struct {
	RuleID        string         `json:"rule_id"`
	Description   string         `json:"description"`
	Confidence    string         `json:"confidence"`
	Match         Match          `json:"match"`
	Attributes    map[string]any `json:"attributes"`
	Location      Location       `json:"location"`
	Analysis      *Analysis      `json:"analysis"`
	ComponentSets []ComponentSet `json:"component_sets"`
	Encodings     []string       `json:"encodings"`
	DecodeDepth   int            `json:"decode_depth"`
	Tags          []string       `json:"tags"`
}

// Match holds the matched text, the selected secret value, named capture
// groups, the surrounding text and the fingerprint.
type Match struct {
	Full        string         `json:"full"`
	Value       string         `json:"value"`
	Captures    map[string]any `json:"captures"`
	Context     string         `json:"context"`
	Fingerprint string         `json:"fingerprint"`
}

// Location is a 1-based position; EndColumn is inclusive.
type Location struct {
	Path        string `json:"path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	StartColumn int    `json:"start_column"`
	EndColumn   int    `json:"end_column"`
}

// Analysis is present only when validation or analysis ran.
type Analysis struct {
	Status         string         `json:"status"`
	StatusReason   string         `json:"status_reason"`
	StatusMetadata map[string]any `json:"status_metadata"`
}

// ComponentSet lists the extra matches a composite rule required.
type ComponentSet struct {
	Components []Component `json:"components"`
}

// Component is one extra match of a composite rule.
type Component struct {
	RuleID   string   `json:"rule_id"`
	Optional bool     `json:"optional"`
	Match    Match    `json:"match"`
	Location Location `json:"location"`
}

// componentProperty is the SARIF shape of a component. Set is the 0-based index
// of the component set it came from. Location keeps betterleaks coordinates
// (1-based, inclusive end column); its path is made relative like the finding path.
type componentProperty struct {
	Set      int      `json:"set"`
	RuleID   string   `json:"rule_id"`
	Optional bool     `json:"optional"`
	Value    string   `json:"value,omitempty"`
	Full     string   `json:"full,omitempty"`
	Location Location `json:"location"`
}

// ParseReport reads and decodes a betterleaks JSON report.
func ParseReport(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read betterleaks report: %w", err)
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse betterleaks report %q: %w", path, err)
	}
	return &report, nil
}

// LoadCompleteReport applies the result handling policy: only exit codes 0 and
// 2 (with --exit-code 2) are accepted, and the report must parse and have
// scan.state == "complete". Partial findings are never returned.
func LoadCompleteReport(exitCode int, path string) (*Report, error) {
	if exitCode != 0 && exitCode != 2 {
		return nil, fmt.Errorf("betterleaks exited with code %d", exitCode)
	}
	report, err := ParseReport(path)
	if err != nil {
		return nil, err
	}
	if report.Scan.State != scanStateComplete {
		return nil, fmt.Errorf("betterleaks scan state is %q, expected %q", report.Scan.State, scanStateComplete)
	}
	return report, nil
}

// JSONToSarif converts a parsed report to SARIF 2.1.0. Absolute finding paths
// inside targetPath are emitted relative to it.
func JSONToSarif(report *Report, targetPath string) (*sarif.Report, error) {
	out, err := sarif.New(sarif.Version210)
	if err != nil {
		return nil, fmt.Errorf("failed to create SARIF report: %w", err)
	}

	run := sarif.NewRunWithInformationURI(toolName, toolInformationURI)
	run.Tool.Driver.WithVersion(report.Scan.BetterleaksVersion)
	for i := range report.Findings {
		addFinding(run, &report.Findings[i], targetPath)
	}

	run.AddInvocation(report.Scan.State == scanStateComplete)
	run.PropertyBag = *sarif.NewPropertyBag()
	run.Add("config_hash", report.Scan.ConfigHash)
	run.Add("bytes_scanned", report.Scan.BytesScanned)
	run.Add("num_findings", report.Scan.NumFindings)

	out.AddRun(run)
	return out, nil
}

func addFinding(run *sarif.Run, f *Finding, targetPath string) {
	rule := run.AddRule(f.RuleID)
	if rule.ShortDescription == nil {
		rule.WithDescription(f.Description).
			WithFullDescription(sarif.NewMultiformatMessageString(f.Description))
		rule.WithProperties(sarif.Properties{"tags": nonNil(f.Tags)})
	}

	result := sarif.NewRuleResult(f.RuleID).
		WithMessage(sarif.NewTextMessage(f.Description)).
		WithLevel(sarifLevel(f.Tags))
	if f.Match.Fingerprint != "" {
		result.WithPartialFingerPrints(map[string]interface{}{fingerprintKey: f.Match.Fingerprint})
	}
	if f.Location.Path != "" {
		result.AddLocation(buildLocation(f, targetPath))
	}
	result.PropertyBag = *sarif.NewPropertyBag()
	setResultProperties(result, f, targetPath)
	run.AddResult(result)
}

func buildLocation(f *Finding, targetPath string) *sarif.Location {
	region := sarif.NewRegion().
		WithStartLine(f.Location.StartLine).
		WithStartColumn(f.Location.StartColumn).
		WithEndLine(f.Location.EndLine).
		WithEndColumn(f.Location.EndColumn + 1)
	if f.Match.Full != "" {
		region.WithSnippet(sarif.NewArtifactContent().WithText(f.Match.Full))
	}
	uri := artifactURI(f.Location.Path, targetPath)
	physical := sarif.NewPhysicalLocation().
		WithArtifactLocation(sarif.NewArtifactLocation().WithUri(uri)).
		WithRegion(region)
	if f.Match.Context != "" {
		// betterleaks does not report the context's line range, so only the text is set.
		physical.WithContextRegion(sarif.NewRegion().WithSnippet(sarif.NewArtifactContent().WithText(f.Match.Context)))
	}
	return sarif.NewLocation().WithPhysicalLocation(physical)
}

// artifactURI makes an absolute path inside targetPath relative to it, as fs
// mode reports absolute paths while other scanners report target-relative ones.
// Relative paths and paths outside the target are returned unchanged.
func artifactURI(path, targetPath string) string {
	if !filepath.IsAbs(path) || targetPath == "" {
		return strings.TrimPrefix(path, "./")
	}
	rel, err := filepath.Rel(filepath.Clean(targetPath), filepath.Clean(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	if rel == "." {
		rel = filepath.Base(path)
	}
	return filepath.ToSlash(rel)
}

func setResultProperties(result *sarif.Result, f *Finding, targetPath string) {
	result.Add("confidence", f.Confidence)
	for _, key := range []string{"resource", "git.sha", "git.date"} {
		if v, ok := f.Attributes[key].(string); ok && v != "" {
			result.Add(key, v)
		}
	}
	if msg, ok := f.Attributes["git.message"].(string); ok && msg != "" {
		result.Add("git.message", msg)
	}
	if f.Match.Value != "" {
		result.Add("secret", f.Match.Value)
	}
	if len(f.Match.Captures) > 0 {
		result.Add("captures", f.Match.Captures)
	}
	if f.Analysis != nil {
		addAnalysisProperties(result, f.Analysis)
	}
	if len(f.Encodings) > 0 {
		result.Add("encodings", f.Encodings)
	}
	if f.DecodeDepth > 0 {
		result.Add("decode_depth", f.DecodeDepth)
	}
	if ids := componentRuleIDs(f.ComponentSets); len(ids) > 0 {
		result.Add("component_rule_ids", ids)
	}
	if comps := componentProperties(f.ComponentSets, targetPath); len(comps) > 0 {
		result.Add("components", comps)
	}
}

func addAnalysisProperties(result *sarif.Result, a *Analysis) {
	if a.Status != "" {
		result.Add("validation_status", a.Status)
	}
	if a.StatusReason != "" {
		result.Add("validation_reason", a.StatusReason)
	}
	if len(a.StatusMetadata) > 0 {
		result.Add("validation_metadata", a.StatusMetadata)
	}
}

func componentProperties(sets []ComponentSet, targetPath string) []componentProperty {
	var out []componentProperty
	for i, set := range sets {
		for _, c := range set.Components {
			loc := c.Location
			loc.Path = artifactURI(loc.Path, targetPath)
			out = append(out, componentProperty{
				Set:      i,
				RuleID:   c.RuleID,
				Optional: c.Optional,
				Value:    c.Match.Value,
				Full:     c.Match.Full,
				Location: loc,
			})
		}
	}
	return out
}

func componentRuleIDs(sets []ComponentSet) []string {
	seen := map[string]struct{}{}
	var ids []string
	for _, set := range sets {
		for _, c := range set.Components {
			if _, dup := seen[c.RuleID]; dup || c.RuleID == "" {
				continue
			}
			seen[c.RuleID] = struct{}{}
			ids = append(ids, c.RuleID)
		}
	}
	return ids
}

// sarifLevel derives the level from a "severity:<x>" rule tag only. Confidence
// is a detection-quality signal and must not drive severity. Unknown or
// missing severity fails closed to "error".
func sarifLevel(tags []string) string {
	for _, tag := range tags {
		sev, ok := strings.CutPrefix(tag, severityTagPrefix)
		if !ok {
			continue
		}
		switch strings.ToLower(sev) {
		case "critical", "high":
			return "error"
		case "medium":
			return "warning"
		case "low":
			return "note"
		}
	}
	return "error"
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// WriteSarif writes the SARIF report next to the JSON report (same base name,
// .sarif extension) and returns its path. The JSON report is left untouched.
func WriteSarif(report *Report, jsonPath, targetPath string) (string, error) {
	sarifReport, err := JSONToSarif(report, targetPath)
	if err != nil {
		return "", err
	}
	outPath := strings.TrimSuffix(jsonPath, ".json") + ".sarif"
	file, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("failed to create SARIF file: %w", err)
	}
	if err := sarifReport.PrettyWrite(file); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("failed to write SARIF report: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("failed to close SARIF file: %w", err)
	}
	return outPath, nil
}
