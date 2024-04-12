/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/spf13/cobra"
)

type RunOptionsReportPatch struct {
	Input                string
	Output               string
	WhenRule             string
	WhenTextContains     []string
	WhenTextNotContains  []string
	WhenLocationContains []string
	SetSeverity          string
	Delete               bool
}

var allArgumentsReportPatch RunOptionsReportPatch

const (
	OperationWhenRule             = "OperationWhenRule"
	OperationWhenTextContains     = "OperationWhenTextContains"
	OperationWhenTextNotContains  = "OperationWhenTextNotContains"
	OperationWhenLocationContains = "OperationWhenLocationContains"
)

type Condition struct {
	Operation string
	Value     string
}

func isResultSpecific(conditions []Condition) bool {
	resultSpecificConditions := []string{OperationWhenTextContains, OperationWhenTextNotContains, OperationWhenLocationContains}
	for _, condition := range conditions {
		for _, op := range resultSpecificConditions {
			if condition.Operation == op {
				return true
			}
		}
	}
	return false
}

func buildConditions() []Condition {
	conditions := []Condition{}
	if len(allArgumentsReportPatch.WhenRule) != 0 {
		conditions = append(conditions, Condition{
			Operation: OperationWhenRule,
			Value:     allArgumentsReportPatch.WhenRule,
		})
	}
	if len(allArgumentsReportPatch.WhenTextContains) != 0 {
		for _, text := range allArgumentsReportPatch.WhenTextContains {
			conditions = append(conditions, Condition{
				Operation: OperationWhenTextContains,
				Value:     text,
			})
		}
	}
	if len(allArgumentsReportPatch.WhenTextNotContains) != 0 {
		for _, text := range allArgumentsReportPatch.WhenTextNotContains {
			conditions = append(conditions, Condition{
				Operation: OperationWhenTextNotContains,
				Value:     text,
			})
		}
	}
	if len(allArgumentsReportPatch.WhenLocationContains) != 0 {
		for _, loc := range allArgumentsReportPatch.WhenLocationContains {
			conditions = append(conditions, Condition{
				Operation: OperationWhenLocationContains,
				Value:     loc,
			})
		}
	}

	return conditions
}

func copyReport(orig *sarif.Report) (*sarif.Report, error) {
	origJSON, err := json.Marshal(orig)
	if err != nil {
		return nil, err
	}

	clone := sarif.Report{}
	if err = json.Unmarshal(origJSON, &clone); err != nil {
		return nil, err
	}

	return &clone, nil
}

func toSarifLevel(severity string) string {
	switch severity {
	case "high":
		return "error"
	case "medium":
		return "warning"
	case "low":
		return "note"
	default:
		return "none"
	}
}

// If any of conditions don't match - returns false
// otherwise returns true
func evalConditionsForRule(rule *sarif.ReportingDescriptor, conditions []Condition) bool {
	for _, condition := range conditions {
		switch condition.Operation {
		case OperationWhenRule:
			if !strings.Contains(rule.ID, condition.Value) {
				return false
			}
		}
	}
	return true
}

func applyRulePatch(rule *sarif.ReportingDescriptor) {
	if len(allArgumentsReportPatch.SetSeverity) > 0 {
		rule.DefaultConfiguration.Level = toSarifLevel(allArgumentsReportPatch.SetSeverity)
	}
}

func patchRules(report *sarif.Report, conditions []Condition) {
	for _, run := range report.Runs {
		for _, rule := range run.Tool.Driver.Rules {
			if !evalConditionsForRule(rule, conditions) {
				continue
			}
			applyRulePatch(rule)
		}
	}
}

// If any of conditions don't match - returns false
// otherwise returns true
func evalConditionsForResult(result *sarif.Result, conditions []Condition) bool {
	for _, condition := range conditions {
		switch condition.Operation {
		case OperationWhenRule:
			if result.RuleID == nil || !strings.Contains(*result.RuleID, condition.Value) {
				return false
			}
		case OperationWhenTextContains:
			if result.Message.Text == nil || !strings.Contains(*result.Message.Text, condition.Value) {
				return false
			}
		case OperationWhenTextNotContains:
			if result.Message.Text != nil && strings.Contains(*result.Message.Text, condition.Value) {
				return false
			}
		case OperationWhenLocationContains:
			for _, location := range result.Locations {
				if location.PhysicalLocation.ArtifactLocation.URI == nil || !strings.Contains(*location.PhysicalLocation.ArtifactLocation.URI, condition.Value) {
					return false
				}
			}
		}
	}

	return true
}

func applyResultPatch(result *sarif.Result) {
	if len(allArgumentsReportPatch.SetSeverity) > 0 {
		*result.Level = toSarifLevel(allArgumentsReportPatch.SetSeverity)
	}
}

func patchResultsForDelete(report *sarif.Report, conditions []Condition) {
	for _, run := range report.Runs {
		results := []*sarif.Result{}
		for _, result := range run.Results {
			if evalConditionsForResult(result, conditions) {
				continue
			}
			results = append(results, result)
		}
		run.Results = results
	}
}

func patchResultsForSet(report *sarif.Report, conditions []Condition) {
	for _, run := range report.Runs {
		for _, result := range run.Results {
			if !evalConditionsForResult(result, conditions) {
				continue
			}
			applyResultPatch(result)
		}
	}
}

func patchResults(report *sarif.Report, conditions []Condition) {
	if allArgumentsReportPatch.Delete {
		patchResultsForDelete(report, conditions)
	} else {
		patchResultsForSet(report, conditions)
	}
}

var execExampleReportPatch = `  # Overwrite severity for specific rule by rule id and results for this rule
  scanio report-patch -i original-report.sarif -o patched-report.sarif --when-rule java/CSRFDisabled --set-severity low

  # Set lower criticality for results of specific rule, when description says that source of malicious data is coming from command line arguments.
  scanio report-patch -i original-report.sarif -o patched-report.sarif --when-rule javascript/SQLInjection --when-text-contains "input from a command line argument" --set-severity low

  # Delete results found in node_modules subfolder
  scanio report-patch -i original-report.sarif -o patched-report.sarif --when-location-contains node_modules/ --delete`

// reportPatchCmd represents the reportPatch command
var reportPatchCmd = &cobra.Command{
	Use:     "report-patch",
	Short:   "This command allows to overwrite some data in sarif report. For example you can decrease severity for some falsy rules, or make some severity adjustments depending on fionding description, finding location and so on.",
	Example: execExampleReportPatch,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("reportPatch called")

		jsonFile, err := os.Open(allArgumentsReportPatch.Input)
		if err != nil {
			return err
		}
		defer jsonFile.Close()

		var originalSarifReport sarif.Report
		byteValue, _ := io.ReadAll(jsonFile)
		json.Unmarshal([]byte(byteValue), &originalSarifReport)

		report, err := copyReport(&originalSarifReport)
		if err != nil {
			return err
		}

		conditions := buildConditions()

		if !isResultSpecific(conditions) {
			patchRules(report, conditions)
		}
		patchResults(report, conditions)

		if len(allArgumentsReportPatch.Output) > 0 {
			data, _ := json.MarshalIndent(report, "", "  ")
			_ = os.WriteFile(allArgumentsReportPatch.Output, data, 0644)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(reportPatchCmd)

	reportPatchCmd.Flags().StringVarP(&allArgumentsReportPatch.Input, "input", "i", "", "input file with SAST report ion SARIF format")
	reportPatchCmd.Flags().StringVarP(&allArgumentsReportPatch.Output, "output", "o", "", "output file where patched SARIF report will be saved to")
	reportPatchCmd.Flags().StringVar(&allArgumentsReportPatch.WhenRule, "when-rule", "", "condition to apply patch only to rules like this (check substring, because for semgrep when use local rules rule name contains full path)")
	reportPatchCmd.Flags().StringArrayVar(&allArgumentsReportPatch.WhenTextContains, "when-text-contains", []string{}, "condition to apply patch only to results with description like this")
	reportPatchCmd.Flags().StringArrayVar(&allArgumentsReportPatch.WhenTextNotContains, "when-text-not-contains", []string{}, "condition to apply patch only to results with description not like this")
	reportPatchCmd.Flags().StringArrayVar(&allArgumentsReportPatch.WhenLocationContains, "when-location-contains", []string{}, "condition to apply patch only to results with specific location")
	reportPatchCmd.Flags().StringVar(&allArgumentsReportPatch.SetSeverity, "set-severity", "", "patch severity level (high, medum, low, none)")
	reportPatchCmd.Flags().BoolVar(&allArgumentsReportPatch.Delete, "delete", false, "delete finding that matches conditions")
}
