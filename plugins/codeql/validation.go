package main

import (
	"fmt"
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/validation"
)

// validateScan checks the necessary fields in ScannerScanRequest and returns errors if they are not set.
func (g *ScannerCodeQL) validateScan(args *shared.ScannerScanRequest) error {
	if err := validation.ValidateScanArgs(args); err != nil {
		return err
	}

	return nil
}

// validateFormatSoft verifies if the given format is supported and logs a warning if it is not.
func (g *ScannerCodeQL) validateFormatSoft(format string) {
	formatList := []string{"csv", "sarif-latest", "sarifv2.1.0", "graphtext", "dgml", "dot"}
	if !shared.IsInList(format, formatList) {
		g.logger.Warn(
			"the used known version of CodeQL doesn't support the passed format type. Continuing scan...",
			"reportFormat", format,
			"supportedFormats", strings.Join(formatList, ", "),
		)
	}
}

// validateLanguageHard checks if the given language is supported by CodeQL.
func validateLanguageHard(language string) error {
	supportedLanguageList := []string{"cpp", "csharp", "go", "java", "javascript", "python", "ruby", "swift"}
	if !shared.IsInList(language, supportedLanguageList) {
		return fmt.Errorf("unsupported language for CodeQL: %s", language)
	}
	return nil
}
