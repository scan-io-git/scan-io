package main

import (
	"github.com/scan-io-git/scan-io/pkg/shared"
)

// supportedReportFormats is a private variable holding the list of supported report formats by the scanner.
var supportedReportFormats = []string{"json", "text", "html"}

// CheckReportFormat determines the report format to use and whether a conversion is needed.
func (g *ScannerTrufflehog3) CheckReportFormat(args *shared.ScannerScanRequest) (originalFormat string, reportFormat string, needsConversion bool) {
	originalFormat = args.ReportFormat

	if originalFormat != "" {
		switch originalFormat {
		case "sarif":
			g.logger.Warn("SARIF report format requested. Default scanner JSON report will be converted.")
			reportFormat = "json"
			needsConversion = true
		case "markdown":
			g.logger.Warn("Human-readable markdown report format requested. Default scanner JSON report will be converted.")
			reportFormat = "json"
			needsConversion = true
		default:
			if isSupportedFormat(originalFormat) {
				reportFormat = originalFormat
			} else {
				g.logger.Warn("Unsupported report format requested. Defaulting to JSON.", "requested_format", originalFormat)
				reportFormat = "json"
			}
		}
	}

	return originalFormat, reportFormat, needsConversion
}

// isSupportedFormat checks if a format is supported by the scanner.
func isSupportedFormat(format string) bool {
	for _, supportedFormat := range supportedReportFormats {
		if format == supportedFormat {
			return true
		}
	}
	return false
}
