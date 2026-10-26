package sarif

import (
	"fmt"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

// MergeReports concatenates the Results of every input report's first run into a
// single run, so a multi-input render still produces exactly one *sarif.Run for the
// template's {{range .Report.Runs}} to walk. Each result's Properties (including
// "Scanner", stamped per input by EnrichResultsMetadataProperty) are carried over
// untouched. Order is preserved: all results from reports[0] first, then reports[1],
// and so on — callers that want a reproducible render across an unordered tool set
// must sort their input paths before calling ReadReport.
//
// Must run after each input's own enrichment chain: that chain is bound to Runs[0]
// and expects a single-tool report, so merging first would corrupt it.
func MergeReports(reports []*Report) (*Report, error) {
	if len(reports) == 0 {
		return nil, fmt.Errorf("no reports to merge")
	}

	first := reports[0]
	merged := &sarif.Run{
		Tool: first.Runs[0].Tool,
	}
	for _, report := range reports {
		merged.Results = append(merged.Results, report.Runs[0].Results...)
	}

	return &Report{
		Report: &sarif.Report{
			Version: first.Report.Version,
			Schema:  first.Report.Schema,
			Runs:    []*sarif.Run{merged},
		},
		logger:       first.logger,
		sourceFolder: first.sourceFolder,
	}, nil
}

// ExtractToolsMetadata collects tool metadata from each report, in the order given.
// Must be called before MergeReports: once merged, a report's own Tool field no
// longer identifies a single scanner.
func ExtractToolsMetadata(reports []*Report) ([]ToolMetadata, error) {
	tools := make([]ToolMetadata, 0, len(reports))
	for _, report := range reports {
		tm, err := report.ExtractToolNameAndVersion()
		if err != nil {
			return nil, err
		}
		tools = append(tools, *tm)
	}
	return tools, nil
}
