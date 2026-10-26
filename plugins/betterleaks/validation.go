package main

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/validation"
)

const (
	commandFS  = "fs"
	commandGit = "git"
)

// resolveCommand returns the betterleaks subcommand, defaulting to "fs".
func resolveCommand(c string) string {
	if c == "" {
		return commandFS
	}
	return c
}

// validateScan checks the request fields and the requested subcommand.
func (g *ScannerBetterleaks) validateScan(args *shared.ScannerScanRequest) error {
	if err := validation.ValidateScanArgs(args); err != nil {
		return err
	}
	switch cmd := resolveCommand(args.Command); cmd {
	case commandFS, commandGit:
		return nil
	default:
		return fmt.Errorf("unsupported --command %q: expected %q or %q", cmd, commandFS, commandGit)
	}
}

// CheckReportFormat determines the format of the betterleaks report to
// produce. betterleaks only writes JSON, so SARIF is a conversion of it.
func (g *ScannerBetterleaks) CheckReportFormat(args *shared.ScannerScanRequest) (originalFormat string, reportFormat string, needsConversion bool) {
	originalFormat = args.ReportFormat

	switch originalFormat {
	case "", "json":
	case "sarif":
		g.logger.Warn("SARIF report format requested. Default scanner JSON report will be converted")
		needsConversion = true
	default:
		g.logger.Warn("Unsupported report format requested. Defaulting to JSON.", "requested_format", originalFormat)
	}

	return originalFormat, "json", needsConversion
}
