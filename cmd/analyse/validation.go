package analyse

import (
	"fmt"
	"os"
)

// validateAnalyseArgs validates the arguments provided to the analyse command.
func validateAnalyseArgs(allArgumentsAnalyse *RunOptionsAnalyse, args []string, argsLenAtDash int) error {
	if allArgumentsAnalyse.Scanner == "" {
		return fmt.Errorf("the 'scanner' flag must be specified")
	}

	if argsLenAtDash > -1 {
		allArgumentsAnalyse.AdditionalArgs = args[argsLenAtDash:]
	}

	if (len(args) == 0 || (len(args) > 0 && argsLenAtDash == 0)) && allArgumentsAnalyse.InputFile == "" {
		return fmt.Errorf("either 'input-file' flag or a target path must be specified")
	}

	// TODO: add checking a format and using input file in case of the right given file as a single file
	if len(args) > 0 && (argsLenAtDash == -1 || argsLenAtDash == 1) {
		if allArgumentsAnalyse.InputFile != "" {
			return fmt.Errorf("you cannot use an 'input-file' flag and a target path at the same time")
		}

		targetPath := args[0]
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return fmt.Errorf("the target path does not exist: %v", targetPath)
		}

		return nil
	}

	if allArgumentsAnalyse.InputFile == "" {
		return fmt.Errorf("the 'input-file' flag must be specified")
	}

	if allArgumentsAnalyse.Threads <= 0 {
		return fmt.Errorf("the 'threads' flag must be a positive integer")
	}

	// TODO: add validation for the input file format
	return nil
}
