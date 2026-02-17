// Package sarifadapter converts SARIF reports into the internal findings model.
//
// V1 scope: only RuleID, Title, and Scanner are populated; all other Finding
// fields are left at zero value (empty string, 0, or nil).
package sarifadapter

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/findings"
)

// FromReport converts a SARIF report into a slice of findings.
// One Finding is produced per result; runs are flattened.
// V1 maps only RuleID, Title, and Scanner; other fields are zero value.
func FromReport(report *sarif.Report) []findings.Finding {
	return convertReport(report)
}

// FromFile reads a SARIF file from path and converts it to findings.
// Returns an error if the file cannot be read or decoded as SARIF.
func FromFile(path string) ([]findings.Finding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report sarif.Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return convertReport(&report), nil
}

// convertReport iterates all runs and results, emitting one Finding per result.
func convertReport(report *sarif.Report) []findings.Finding {
	if report == nil {
		return nil
	}
	out := []findings.Finding{}
	for _, run := range report.Runs {
		scannerName := ""
		if run.Tool.Driver != nil {
			scannerName = strings.TrimSpace(run.Tool.Driver.Name)
		}
		rulesByID := rulesByID(run)
		for _, result := range run.Results {
			out = append(out, resultToFinding(result, rulesByID, scannerName))
		}
	}
	return out
}

func rulesByID(run *sarif.Run) map[string]*sarif.ReportingDescriptor {
	m := make(map[string]*sarif.ReportingDescriptor)
	if run == nil || run.Tool.Driver == nil {
		return m
	}
	for _, r := range run.Tool.Driver.Rules {
		if r == nil {
			continue
		}
		id := strings.TrimSpace(r.ID)
		if id != "" {
			m[id] = r
		}
	}
	return m
}

func resultToFinding(res *sarif.Result, rulesByID map[string]*sarif.ReportingDescriptor, scannerName string) findings.Finding {
	f := findings.Finding{}
	if res == nil {
		return f
	}
	if res.RuleID != nil {
		f.RuleID = strings.TrimSpace(*res.RuleID)
	}
	rule, _ := rulesByID[f.RuleID]
	f.Title = titleFromRule(rule, f.RuleID)
	f.Scanner = scannerName
	return f
}

// titleFromRule returns title: ShortDescription.Text, else Name, else rule ID.
func titleFromRule(rule *sarif.ReportingDescriptor, ruleID string) string {
	if rule != nil {
		if rule.ShortDescription != nil && rule.ShortDescription.Text != nil {
			if t := strings.TrimSpace(*rule.ShortDescription.Text); t != "" {
				return t
			}
		}
		if rule.Name != nil {
			if t := strings.TrimSpace(*rule.Name); t != "" {
				return t
			}
		}
		return strings.TrimSpace(rule.ID)
	}
	return ruleID
}
