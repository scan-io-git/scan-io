package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

// Trufflehog3Issue represents a single issue found by Trufflehog3.
type Trufflehog3Issue struct {
	Rule   *Trufflehog3Rule `json:"rule"`
	Path   string           `json:"path"`
	Line   string           `json:"line"`
	Secret string           `json:"secret"`
	ID     string           `json:"id,omitempty"`
	Branch string           `json:"branch,omitempty"`
	Commit string           `json:"commit,omitempty"`
	Author string           `json:"author,omitempty"`
	Date   string           `json:"date,omitempty"`
}

// Trufflehog3Rule represents the rule that triggered an issue.
type Trufflehog3Rule struct {
	ID       string `json:"id"`
	Message  string `json:"message"`
	Pattern  string `json:"pattern"`
	Severity string `json:"severity"`
}

// Trufflehog3Report represents a collection of Trufflehog3 issues.
type Trufflehog3Report []*Trufflehog3Issue

// Deduplicate removes duplicate issues from the report.
func (report Trufflehog3Report) Deduplicate() Trufflehog3Report {
	seen := make(map[string]struct{})
	var deduplicated Trufflehog3Report

	for _, issue := range report {
		uniqueKey := fmt.Sprintf("%s:%s:%s:%s", issue.ID, issue.Path, issue.Line, issue.Secret)
		if _, exists := seen[uniqueKey]; !exists {
			seen[uniqueKey] = struct{}{}
			deduplicated = append(deduplicated, issue)
		}
	}

	return deduplicated
}

// Render produces a human-readable report of the Trufflehog3 issues.
func (report Trufflehog3Report) Render() string {
	triggers := make(map[string]Trufflehog3Report)
	for _, issue := range report {
		triggers[issue.Path] = append(triggers[issue.Path], issue)
	}

	var output strings.Builder
	for filePath, issues := range triggers {
		sort.Slice(issues, func(i, j int) bool {
			line1, _ := strconv.Atoi(issues[i].Line)
			line2, _ := strconv.Atoi(issues[j].Line)
			return line1 < line2
		})

		output.WriteString(fmt.Sprintf("#### Path: %v\n```\n", filePath))
		for _, issue := range issues {
			output.WriteString(fmt.Sprintf("    %s (%s severity) line %s: %s\n\n",
				issue.Rule.ID, issue.Rule.Severity, issue.Line, issue.Secret))
		}
		output.WriteString("```\n")
	}
	return output.String()
}

// JsonToPlainReport converts a Trufflehog3 JSON report to a plain text format.
func JsonToPlainReport(filePath string) (string, error) {
	var (
		reportBuilder     strings.Builder
		reportTrufflehog3 Trufflehog3Report
	)

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return filePath, fmt.Errorf("error opening file: %v", err)
	}

	if err := json.Unmarshal(fileContent, &reportTrufflehog3); err != nil {
		return filePath, fmt.Errorf("error parsing report: %v", err)
	}

	reportTrufflehog3 = reportTrufflehog3.Deduplicate()

	reportBuilder.WriteString("### Trufflehog3 scanner resutls\n")
	reportBuilder.WriteString(fmt.Sprintf("The scanner found %d issues.\n\n", len(reportTrufflehog3)))
	reportBuilder.WriteString(reportTrufflehog3.Render())
	outputFilePath := strings.TrimSuffix(filePath, ".json") + ".markdown"
	if err := os.WriteFile(outputFilePath, []byte(reportBuilder.String()), 0644); err != nil {
		return outputFilePath, fmt.Errorf("error writing plain text report: %v", err)
	}

	return outputFilePath, nil
}

// JsonToSarifReport converts a Trufflehog3 JSON report to SARIF format.
func JsonToSarifReport(filePath string) (string, error) {
	var reportTrufflehog3 Trufflehog3Report

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return filePath, fmt.Errorf("error opening file: %v", err)
	}

	if err := json.Unmarshal(fileContent, &reportTrufflehog3); err != nil {
		return filePath, fmt.Errorf("failed to unmarshal JSON report: %v", err)
	}
	reportTrufflehog3 = reportTrufflehog3.Deduplicate()
	reportSarif, err := sarif.New(sarif.Version210)
	if err != nil {
		return filePath, fmt.Errorf("failed to create SARIF report: %w", err)
	}

	run := sarif.NewRunWithInformationURI("TruffleHog3", "https://github.com/feeltheajf/trufflehog3")
	for _, issue := range reportTrufflehog3 {
		rule := run.AddRule(issue.Rule.ID).
			WithDescription(issue.Rule.Message).
			WithDefaultConfiguration(&sarif.ReportingConfiguration{
				Level: toSarifErrorLevel(issue.Rule.Severity),
			})

		lineNumber, err := strconv.Atoi(issue.Line)
		if err != nil {
			lineNumber = 0
		}
		location := sarif.NewLocation().WithPhysicalLocation(
			sarif.NewPhysicalLocation().
				WithArtifactLocation(sarif.NewArtifactLocation().WithUri(issue.Path)).
				WithRegion(sarif.NewRegion().WithStartLine(lineNumber)),
		)

		result := sarif.NewRuleResult(rule.ID).
			WithMessage(sarif.NewTextMessage(issue.Rule.Message)).
			WithLevel(toSarifErrorLevel(issue.Rule.Severity)).
			WithLocations([]*sarif.Location{location})
		run.AddResult(result)
	}
	reportSarif.AddRun(run)

	outputFilePath := strings.TrimSuffix(filePath, ".json") + ".sarif"
	file, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return outputFilePath, fmt.Errorf("error writing SARIF report: %v", err)
	}
	defer func() { _ = file.Close() }()
	if err := reportSarif.PrettyWrite(file); err != nil {
		return filePath, err
	}
	return filePath, nil
}

func toSarifErrorLevel(severity string) string {
	switch strings.ToUpper(severity) {
	case "CRITICAL", "HIGH":
		return "error"
	case "MEDIUM":
		return "warning"
	case "LOW", "UNKNOWN":
		return "note"
	default:
		return "none"
	}
}
