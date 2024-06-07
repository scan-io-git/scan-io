package main

import (
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/validation"
)

// validateScan checks the necessary fields in ScannerScanRequest and returns errors if they are not set.
func (g *ScannerBandit) validateScan(args *shared.ScannerScanRequest) error {
	if err := validation.ValidateScanArgs(args); err != nil {
		return err
	}
	g.validateFormatSoft(args.ReportFormat)
	return nil
}

// validateFormatSoft verifies if the given format is supported and logs a warning if it is not.
func (g *ScannerBandit) validateFormatSoft(format string) {
	formatList := []string{"csv", "custom", "html", "json", "screen", "txt", "xml", "yaml"}
	if !shared.IsInList(format, formatList) {
		g.logger.Warn(
			"the used known version of Bandit doesn't support the passed format type. Continue scan...",
			"reportFormat", format,
			"supportedFormats", strings.Join(formatList, ", "),
		)
	}
}
