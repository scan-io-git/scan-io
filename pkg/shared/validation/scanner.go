package validation

import (
	"fmt"
	"os"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

// ValidateScanArgs checks the necessary fields in ScannerScanRequest and returns errors if they are not set
func ValidateScanArgs(args *shared.ScannerScanRequest) error {
	if args.TargetPath == "" {
		return fmt.Errorf("target path is required")
	}

	if args.ResultsPath == "" {
		return fmt.Errorf("results path is required")
	}

	expandedPath, err := files.ExpandPath(args.TargetPath)
	if err != nil {
		return fmt.Errorf("failed to expand path %q: %w", expandedPath, err)
	}
	args.TargetPath = expandedPath

	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return fmt.Errorf("target path does not exist: %q", expandedPath)
	}

	return nil
}
