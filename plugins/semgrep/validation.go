package main

import (
	"fmt"
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/validation"
)

// validateScan checks the necessary fields in ScannerScanRequest and returns errors if they are not set.
func (g *ScannerSemgrep) validateScan(args *shared.ScannerScanRequest) error {
	if err := validation.ValidateScanArgs(args); err != nil {
		return err
	}
	if args.Command != "" && args.Command != ciCommand {
		return fmt.Errorf("unsupported command %q: only %q is supported", args.Command, ciCommand)
	}
	return nil
}

// validateFormatSoft verifies if the given format is supported and logs a warning if it is not.
func (g *ScannerSemgrep) validateFormatSoft(format string) {
	formatList := []string{"json", "junit-xml", "sarif", "text", "vim"}
	if !shared.IsInList(format, formatList) {
		g.logger.Warn(
			"the used known version of Semgrep doesn't support the passed format type. Continuing scan...",
			"reportFormat", format,
			"supportedFormats", strings.Join(formatList, ", "),
		)
	}
}
