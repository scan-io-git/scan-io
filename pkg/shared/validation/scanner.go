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

	paths := map[string]string{
		"target path":  args.TargetPath,
		"results path": args.ResultsPath,
	}

	for name, path := range paths {
		expandedPath, err := files.ExpandPath(path)
		if err != nil {
			return fmt.Errorf("failed to expand path '%s': %w", path, err)
		}

		if name == "results path" {
			if err := files.CreateFolderIfNotExists(expandedPath); err != nil {
				return fmt.Errorf("failed to create results path '%s': %w", expandedPath, err)
			}
		} else {
			if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
				return fmt.Errorf("%s does not exist: %s", name, expandedPath)
			}
		}
	}

	return nil
}
