package main

import (
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/validation"
)

// validateScan checks the necessary fields in ScannerScanRequest and returns errors if they are not set.
func (g *ScannerSemgrep) validateScan(args *shared.ScannerScanRequest) error {
	if err := validation.ValidateScanArgs(args); err != nil {
		return err
	}

	return nil
}
