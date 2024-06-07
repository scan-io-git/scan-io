package main

import (
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/validation"
)

const (
	jsonFormat = "json"
	noneFormat = ""
)

// validateScan checks the necessary fields in ScannerScanRequest and returns errors if they are not set.
func (g *ScannerTrufflehog) validateScan(args *shared.ScannerScanRequest) error {
	if err := validation.ValidateScanArgs(args); err != nil {
		return err
	}
	g.validateFormatSoft(args.ReportFormat)
	return nil
}

// validateFormatSoft verifies if the given format is either empty or "json".
func (g *ScannerTrufflehog) validateFormatSoft(format string) {
	if format != noneFormat && format != jsonFormat {
		g.logger.Warn("the current known version of trufflehog supports only json and none as report formats", "reportFormat", format)
	}
}
