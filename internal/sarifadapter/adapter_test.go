package sarifadapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/findings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromReport_HappyPath_OneRunOneResult(t *testing.T) {
	ruleID := "rule-1"
	shortDesc := "Short title"
	name := "Rule Name"
	rule := &gosarif.ReportingDescriptor{
		ID:               ruleID,
		ShortDescription: &gosarif.MultiformatMessageString{Text: &shortDesc},
		Name:             &name,
	}
	result := &gosarif.Result{RuleID: &ruleID}
	report := &gosarif.Report{
		Version: string(gosarif.Version210),
		Runs: []*gosarif.Run{{
			Tool: gosarif.Tool{
				Driver: &gosarif.ToolComponent{
					Name:  "Semgrep",
					Rules: []*gosarif.ReportingDescriptor{rule},
				},
			},
			Results: []*gosarif.Result{result},
		}},
	}

	got := FromReport(report)
	require.Len(t, got, 1)
	f := got[0]
	assert.Equal(t, ruleID, f.RuleID)
	assert.Equal(t, shortDesc, f.Title)
	assert.Equal(t, "Semgrep", f.Scanner)
	assertV1ZeroFields(t, f)
}

func TestFromReport_MultiRunMultiResult_Flattening(t *testing.T) {
	r1 := "R1"
	r2 := "R2"
	report := &gosarif.Report{
		Version: string(gosarif.Version210),
		Runs: []*gosarif.Run{
			{
				Tool: gosarif.Tool{
					Driver: &gosarif.ToolComponent{
						Name:  "ToolA",
						Rules: []*gosarif.ReportingDescriptor{{ID: r1}},
					},
				},
				Results: []*gosarif.Result{{RuleID: &r1}, {RuleID: &r1}},
			},
			{
				Tool: gosarif.Tool{
					Driver: &gosarif.ToolComponent{
						Name:  "ToolB",
						Rules: []*gosarif.ReportingDescriptor{{ID: r2}},
					},
				},
				Results: []*gosarif.Result{{RuleID: &r2}},
			},
		},
	}

	got := FromReport(report)
	require.Len(t, got, 3)
	assert.Equal(t, "ToolA", got[0].Scanner)
	assert.Equal(t, "ToolA", got[1].Scanner)
	assert.Equal(t, "ToolB", got[2].Scanner)
	assert.Equal(t, r1, got[0].RuleID)
	assert.Equal(t, r1, got[1].RuleID)
	assert.Equal(t, r2, got[2].RuleID)
}

func TestFromReport_MissingOptionalFields_SafeDefaults(t *testing.T) {
	ruleID := "x"
	report := &gosarif.Report{
		Version: string(gosarif.Version210),
		Runs: []*gosarif.Run{{
			Tool:    gosarif.Tool{Driver: nil},
			Results: []*gosarif.Result{{RuleID: &ruleID}},
		}},
	}
	got := FromReport(report)
	require.Len(t, got, 1)
	f := got[0]
	assert.Equal(t, ruleID, f.RuleID)
	assert.Empty(t, f.Scanner)
	assert.Equal(t, ruleID, f.Title)
	assertV1ZeroFields(t, f)
}

func TestFromReport_NilResult_SafeDefault(t *testing.T) {
	report := &gosarif.Report{
		Version: string(gosarif.Version210),
		Runs: []*gosarif.Run{{
			Tool: gosarif.Tool{
				Driver: &gosarif.ToolComponent{Name: "Tool"},
			},
			Results: []*gosarif.Result{nil},
		}},
	}
	got := FromReport(report)
	require.Len(t, got, 1)
	assertV1ZeroFields(t, got[0])
}

func TestFromReport_TitleFallbackOrder(t *testing.T) {
	ruleID := "id"
	shortDesc := "Short"
	name := "Name"
	rule := &gosarif.ReportingDescriptor{
		ID:               ruleID,
		ShortDescription: &gosarif.MultiformatMessageString{Text: &shortDesc},
		Name:             &name,
	}
	result := &gosarif.Result{RuleID: &ruleID}
	report := &gosarif.Report{
		Version: string(gosarif.Version210),
		Runs: []*gosarif.Run{{
			Tool: gosarif.Tool{
				Driver: &gosarif.ToolComponent{
					Name:  "T",
					Rules: []*gosarif.ReportingDescriptor{rule},
				},
			},
			Results: []*gosarif.Result{result},
		}},
	}
	got := FromReport(report)
	require.Len(t, got, 1)
	assert.Equal(t, shortDesc, got[0].Title)

	// No ShortDescription: fallback to Name
	rule.ShortDescription = nil
	got = FromReport(report)
	assert.Equal(t, name, got[0].Title)

	// No Name: fallback to ID
	rule.Name = nil
	got = FromReport(report)
	assert.Equal(t, ruleID, got[0].Title)
}

func TestFromReport_NilReport_ReturnsNil(t *testing.T) {
	got := FromReport(nil)
	assert.Nil(t, got)
}

func TestFromReport_EmptyRuns_ReturnsEmptySlice(t *testing.T) {
	report := &gosarif.Report{Version: string(gosarif.Version210), Runs: []*gosarif.Run{}}
	got := FromReport(report)
	require.NotNil(t, got)
	assert.Empty(t, got)
}

func TestFromFile_ReadsAndConverts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.sarif")
	report := &gosarif.Report{
		Version: string(gosarif.Version210),
		Runs: []*gosarif.Run{{
			Tool: gosarif.Tool{
				Driver: &gosarif.ToolComponent{
					Name: "TestTool",
					Rules: []*gosarif.ReportingDescriptor{{
						ID:               "R1",
						ShortDescription: &gosarif.MultiformatMessageString{Text: ptr("Test rule")},
					}},
				},
			},
			Results: []*gosarif.Result{{RuleID: ptr("R1")}},
		}},
	}
	data, err := json.Marshal(report)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
	got, err := FromFile(path)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "R1", got[0].RuleID)
	assert.Equal(t, "Test rule", got[0].Title)
	assert.Equal(t, "TestTool", got[0].Scanner)
}

func ptr(s string) *string { return &s }

func TestFromFile_NoSuchFile_ReturnsError(t *testing.T) {
	_, err := FromFile(filepath.Join(t.TempDir(), "nonexistent.sarif"))
	require.Error(t, err)
}

func assertV1ZeroFields(t *testing.T, f findings.Finding) {
	t.Helper()
	assert.Empty(t, f.Description, "v1: Description must be zero")
	assert.Empty(t, f.Severity, "v1: Severity must be zero")
	assert.Empty(t, f.FilePath, "v1: FilePath must be zero")
	assert.Zero(t, f.StartLine, "v1: StartLine must be 0")
	assert.Zero(t, f.EndLine, "v1: EndLine must be 0")
	assert.Empty(t, f.Tags, "v1: Tags must be empty")
	assert.Empty(t, f.References, "v1: References must be empty")
	assert.Empty(t, f.Properties, "v1: Properties must be empty")
	assert.Empty(t, f.CodeFlows, "v1: CodeFlows must be empty")
}
