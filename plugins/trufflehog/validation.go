package main

import (
	"strings"

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
	return nil
}

// validateFormatSoft verifies if the given format is supported and logs a warning if it is not.
func (g *ScannerTrufflehog) validateFormatSoft(format string) {
	formatList := []string{"json"}
	if !shared.IsInList(format, formatList) {
		g.logger.Warn(
			"the used version of Trufflehog doesn't support the passed format type. Continuing scan...",
			"reportFormat", format,
			"supportedFormats", strings.Join(formatList, ", "),
		)
	}
}
