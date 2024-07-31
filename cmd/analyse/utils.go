package analyse

import (
	"fmt"

	"github.com/scan-io-git/scan-io/pkg/shared"

	utils "github.com/scan-io-git/scan-io/internal/utils"
)

// Mode constants
const (
	ModeSinglePath = "single-path"
	ModeInputFile  = "input-file"
)

// determineMode determines the mode based on the provided arguments.
func determineMode(args []string, argsLenAtDash int) string {
	if len(args) > 0 && (argsLenAtDash == -1 || argsLenAtDash == 1) {
		return ModeSinglePath
	}
	return ModeInputFile
}

// prepareScanTargets prepares the targets for scanning based on the validated arguments.
func prepareScanTargets(options *RunOptionsAnalyse, args []string, mode string) ([]shared.RepositoryParams, string, error) {
	var reposInf []shared.RepositoryParams
	var targetPath string

	switch mode {
	case ModeSinglePath:
		targetPath = args[0]
	case ModeInputFile:
		reposData, err := utils.ReadReposFile2(options.InputFile)
		if err != nil {
			return nil, "", fmt.Errorf("error parsing the input file %s: %v", options.InputFile, err)
		}
		reposInf = reposData
	default:
		return nil, "", fmt.Errorf("invalid analysing mode: %s", mode)
	}

	return reposInf, targetPath, nil
}
