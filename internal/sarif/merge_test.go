package sarif

import (
	"testing"

	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
)

func TestMergeReports_ResultCountAndOrderPreserved(t *testing.T) {
	id := "rule.test"
	r1a := resultFor(id)
	r1a.Properties = map[string]any{"Scanner": "Tool A", "seq": "1a"}
	r1b := resultFor(id)
	r1b.Properties = map[string]any{"Scanner": "Tool A", "seq": "1b"}
	report1 := makeSimpleReport(id, ruleWithTags(id), r1a)
	report1.Runs[0].Results = append(report1.Runs[0].Results, r1b)

	r2a := resultFor(id)
	r2a.Properties = map[string]any{"Scanner": "Tool B", "seq": "2a"}
	report2 := makeSimpleReport(id, ruleWithTags(id), r2a)

	merged, err := MergeReports([]*Report{&report1, &report2})
	if err != nil {
		t.Fatalf("MergeReports returned error: %v", err)
	}

	if got := len(merged.Runs); got != 1 {
		t.Fatalf("merged run count = %d, want 1", got)
	}

	results := merged.Runs[0].Results
	if got := len(results); got != 3 {
		t.Fatalf("merged result count = %d, want 3", got)
	}

	wantSeq := []string{"1a", "1b", "2a"}
	for i, want := range wantSeq {
		got, _ := results[i].Properties["seq"].(string)
		if got != want {
			t.Errorf("results[%d].seq = %q, want %q (order not preserved)", i, got, want)
		}
	}
}

func TestMergeReports_ScannerPropertyIntact(t *testing.T) {
	id := "rule.test"
	rA := resultFor(id)
	rA.Properties = map[string]any{"Scanner": "Tool A"}
	reportA := makeSimpleReport(id, ruleWithTags(id), rA)

	rB := resultFor(id)
	rB.Properties = map[string]any{"Scanner": "Tool B"}
	reportB := makeSimpleReport(id, ruleWithTags(id), rB)

	merged, err := MergeReports([]*Report{&reportA, &reportB})
	if err != nil {
		t.Fatalf("MergeReports returned error: %v", err)
	}

	results := merged.Runs[0].Results
	if got, _ := results[0].Properties["Scanner"].(string); got != "Tool A" {
		t.Errorf("results[0].Scanner = %q, want %q", got, "Tool A")
	}
	if got, _ := results[1].Properties["Scanner"].(string); got != "Tool B" {
		t.Errorf("results[1].Scanner = %q, want %q", got, "Tool B")
	}
}

func TestMergeReports_SingleInputPassthrough(t *testing.T) {
	id := "rule.test"
	r1 := resultFor(id)
	r1.Properties = map[string]any{"Scanner": "Tool A"}
	r2 := resultFor(id)
	r2.Properties = map[string]any{"Scanner": "Tool A"}
	report := makeSimpleReport(id, ruleWithTags(id), r1)
	report.Runs[0].Results = append(report.Runs[0].Results, r2)

	merged, err := MergeReports([]*Report{&report})
	if err != nil {
		t.Fatalf("MergeReports returned error: %v", err)
	}

	got := merged.Runs[0].Results
	want := report.Runs[0].Results
	if len(got) != len(want) {
		t.Fatalf("result count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("results[%d] pointer differs from original single-input result", i)
		}
	}
}

func TestMergeReports_EmptySliceErrors(t *testing.T) {
	if _, err := MergeReports(nil); err == nil {
		t.Error("MergeReports(nil) = nil error, want error")
	}
	if _, err := MergeReports([]*Report{}); err == nil {
		t.Error("MergeReports(empty slice) = nil error, want error")
	}
}

func TestMergeReports_InputWithZeroResults(t *testing.T) {
	id := "rule.test"
	empty := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{Name: "Empty Scanner"},
					},
					Results: nil,
				},
			},
		},
	}

	r := resultFor(id)
	r.Properties = map[string]any{"Scanner": "Tool A"}
	report := makeSimpleReport(id, ruleWithTags(id), r)

	merged, err := MergeReports([]*Report{&empty, &report})
	if err != nil {
		t.Fatalf("MergeReports returned error: %v", err)
	}

	if got := len(merged.Runs[0].Results); got != 1 {
		t.Fatalf("merged result count = %d, want 1", got)
	}
}

func TestExtractToolsMetadata(t *testing.T) {
	id := "rule.test"
	r1 := resultFor(id)
	report1 := makeSimpleReport(id, ruleWithTags(id), r1)

	version := "1.2.3"
	report2 := makeSimpleReport(id, ruleWithTags(id), resultFor(id))
	report2.Runs[0].Tool.Driver.Name = "Other Scanner"
	report2.Runs[0].Tool.Driver.SemanticVersion = &version

	tools, err := ExtractToolsMetadata([]*Report{&report1, &report2})
	if err != nil {
		t.Fatalf("ExtractToolsMetadata returned error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("tools count = %d, want 2", len(tools))
	}
	if tools[0].Name != "Test Scanner" {
		t.Errorf("tools[0].Name = %q, want %q", tools[0].Name, "Test Scanner")
	}
	if tools[1].Name != "Other Scanner" {
		t.Errorf("tools[1].Name = %q, want %q", tools[1].Name, "Other Scanner")
	}
	if tools[1].Version == nil || *tools[1].Version != version {
		t.Errorf("tools[1].Version = %v, want %q", tools[1].Version, version)
	}
}
