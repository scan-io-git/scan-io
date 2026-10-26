package sarif

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	gosarif "github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

func TestEnrichResultsLevelPropertyInitialisesResultProperties(t *testing.T) {
	ruleID := "CODEQL-0001"

	rule := &gosarif.ReportingDescriptor{
		ID: ruleID,
		Properties: gosarif.Properties{
			"problem.severity": "warning",
		},
	}

	result := &gosarif.Result{
		RuleID: &ruleID,
	}

	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Name:  "CodeQL",
							Rules: []*gosarif.ReportingDescriptor{rule},
						},
					},
					Results: []*gosarif.Result{result},
				},
			},
		},
	}

	report.EnrichResultsLevelProperty()

	if result.Properties == nil {
		t.Fatalf("expected result properties to be initialised, but it was nil")
	}

	level, ok := result.Properties["Level"]
	if !ok {
		t.Fatalf("expected Level property to be set on result properties")
	}

	if level != "warning" {
		t.Fatalf("expected Level property to be %q, got %v", "warning", level)
	}
}

func TestEnrichResultsLevelPropertyHandlesMultipleRuns(t *testing.T) {
	ruleIDOne := "RULE-ONE"
	ruleIDTwo := "RULE-TWO"
	resultLevel := "note"

	runOneRule := gosarif.NewRule(ruleIDOne).WithProperties(gosarif.Properties{
		"problem.severity": "warning",
	})
	runTwoRule := gosarif.NewRule(ruleIDTwo)

	runOneResult := &gosarif.Result{
		RuleID: &ruleIDOne,
	}
	runTwoResult := &gosarif.Result{
		RuleID: &ruleIDTwo,
		Level:  &resultLevel,
	}

	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Name:  "ToolOne",
							Rules: []*gosarif.ReportingDescriptor{runOneRule},
						},
					},
					Results: []*gosarif.Result{runOneResult},
				},
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Name:  "ToolTwo",
							Rules: []*gosarif.ReportingDescriptor{runTwoRule},
						},
					},
					Results: []*gosarif.Result{runTwoResult},
				},
			},
		},
	}

	report.EnrichResultsLevelProperty()

	if runOneResult.Properties == nil {
		t.Fatalf("expected runOneResult properties to be initialised")
	}
	if lvl := runOneResult.Properties["Level"]; lvl != "warning" {
		t.Fatalf("expected runOneResult level to be %q, got %v", "warning", lvl)
	}

	if runTwoResult.Properties == nil {
		t.Fatalf("expected runTwoResult properties to be initialised")
	}
	if lvl := runTwoResult.Properties["Level"]; lvl != "note" {
		t.Fatalf("expected runTwoResult level to be %q, got %v", "note", lvl)
	}
}

func TestEnrichResultsLevelPropertyUsesDefaultConfigurationLevel(t *testing.T) {
	ruleID := "RULE-DEFAULT"
	rule := gosarif.NewRule(ruleID)
	rule.DefaultConfiguration = gosarif.NewReportingConfiguration().WithLevel("error")

	result := &gosarif.Result{
		RuleID: &ruleID,
	}

	report := Report{
		Report: &gosarif.Report{
			Version: string(gosarif.Version210),
			Runs: []*gosarif.Run{
				{
					Tool: gosarif.Tool{
						Driver: &gosarif.ToolComponent{
							Name:  "Tool",
							Rules: []*gosarif.ReportingDescriptor{rule},
						},
					},
					Results: []*gosarif.Result{result},
				},
			},
		},
	}

	report.EnrichResultsLevelProperty()

	if result.Properties == nil {
		t.Fatalf("expected result properties to be initialised")
	}
	level, ok := result.Properties["Level"].(string)
	if !ok {
		t.Fatalf("expected Level property to be a string, got %T", result.Properties["Level"])
	}
	if level != "error" {
		t.Fatalf("expected Level property to be %q, got %q", "error", level)
	}
}

func TestReadLineFromFileConfinedToSourceFolder(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.py"), []byte("first\nsecond\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "credentials")
	if err := os.WriteFile(secret, []byte("aws_secret_access_key=example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "config.py")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		uri         string
		want        string
		wantOutside bool
	}{
		{name: "relative path", uri: "app.py", want: "second"},
		{name: "absolute path inside source folder", uri: filepath.Join(root, "app.py"), want: "second"},
		{name: "symlink to outside file", uri: "config.py", wantOutside: true},
		{name: "absolute path outside source folder", uri: secret, wantOutside: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Report{sourceFolder: root, logger: hclog.NewNullLogger()}
			uri, line := tt.uri, 2
			loc := &gosarif.PhysicalLocation{
				ArtifactLocation: &gosarif.ArtifactLocation{URI: &uri},
				Region:           &gosarif.Region{StartLine: &line},
			}
			got, err := report.readLineFromFile(loc)
			if tt.wantOutside {
				if !errors.Is(err, files.ErrOutsideRoot) {
					t.Fatalf("err = %v, want ErrOutsideRoot", err)
				}
				if got != "" {
					t.Fatalf("returned content %q for refused path", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("line = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnrichResultsLocationPropertySkipsCodeForEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	secret := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(secret, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "config.py")); err != nil {
		t.Fatal(err)
	}
	report := Report{sourceFolder: root, logger: hclog.NewNullLogger()}
	uri, line := "config.py", 1
	loc := &gosarif.Location{PhysicalLocation: &gosarif.PhysicalLocation{
		ArtifactLocation: &gosarif.ArtifactLocation{URI: &uri},
		Region:           &gosarif.Region{StartLine: &line},
	}}

	if err := report.EnrichResultsLocationProperty(loc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := loc.PhysicalLocation.ArtifactLocation.Properties["Code"]; ok {
		t.Fatal("Code property set for a symlink leaving the source folder")
	}
}
