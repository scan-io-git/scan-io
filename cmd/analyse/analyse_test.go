package analyse

import (
	"os"
	"testing"

	// "github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/stretchr/testify/assert"
	// utils "github.com/scan-io-git/scan-io/internal/utils"
)

// type MockRepoFileReader struct{}

// Mock utility function to simulate reading a repositories file
// func (m *MockRepoFileReader) ReadReposFile(inputFile string) ([]shared.RepositoryParams, error) {
// 	return []shared.RepositoryParams{}, nil
// }

func TestValidateAnalyseArgs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scanio_example")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// TODO: add tests for the file format validation
	tmpFile, err := os.CreateTemp(tmpDir, "scanio_testfile")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tests := []struct {
		name          string
		options       RunOptionsAnalyse
		args          []string
		argsLenAtDash int
		wantMode      string
		wantErr       string
	}{
		{
			// valid: scanio analyse --scanner semgrep /path/to/target
			name: "Valid scanner and target path",
			options: RunOptionsAnalyse{
				Scanner: "semgrep",
			},
			args:          []string{tmpDir},
			argsLenAtDash: -1,
			wantMode:      ModeSinglePath,
			wantErr:       "",
		},
		// {
		// 	// valid: scanio analyse --scanner semgrep --input-file /path/to/input.file
		// 	name: "Valid scanner and input file",
		// 	options: RunOptionsAnalyse{
		// 		ScannerPluginName: "semgrep",
		// 		InputFile:         tmpFile.Name(),
		// 	},
		// 	args:          []string{},
		// 	argsLenAtDash: -1,
		// 	wantMode:      ModeInputFile,
		// 	wantErr:       "",
		// },
		// {
		// 	// valid: scanio analyse --scanner semgrep --input-file /path/to/input.file -- --verbose --severity INFO
		// 	name: "Valid scanner with input file and additional args",
		// 	options: RunOptionsAnalyse{
		// 		ScannerPluginName: "semgrep",
		// 		InputFile:         tmpFile.Name(),
		// 	},
		// 	args:          []string{"--", "--verbose", "--severity", "INFO"},
		// 	argsLenAtDash: 0,
		// 	wantMode:      ModeInputFile,
		// 	wantErr:       "",
		// },
		{
			// valid: scanio analyse --scanner semgrep /path/to/target -- --verbose --severity INFO
			name: "Valid scanner with target path and additional args",
			options: RunOptionsAnalyse{
				Scanner: "semgrep",
			},
			args:          []string{tmpDir, "--", "--verbose", "--severity", "INFO"},
			argsLenAtDash: -1,
			wantMode:      ModeSinglePath,
			wantErr:       "",
		},
		{
			// fail: scanio analyse /path/to/target
			name: "Missing scanner flag",
			options: RunOptionsAnalyse{
				InputFile: tmpFile.Name(),
			},
			args:          []string{},
			argsLenAtDash: -1,
			wantMode:      "",
			wantErr:       "the 'scanner' flag must be specified",
		},
		{
			// fail: scanio analyse --scanner semgrep
			name: "Missing both input file and target path",
			options: RunOptionsAnalyse{
				Scanner: "semgrep",
			},
			args:          []string{},
			argsLenAtDash: -1,
			wantMode:      "",
			wantErr:       "either 'input-file' flag or a target path must be specified",
		},
		{
			// fail: scanio analyse --scanner semgrep --input-file /path/to/input.file /path/to/target
			name: "Both input file and target path provided",
			options: RunOptionsAnalyse{
				Scanner:   "semgrep",
				InputFile: tmpFile.Name(),
			},
			args:          []string{tmpDir},
			argsLenAtDash: -1,
			wantMode:      "",
			wantErr:       "you cannot use an 'input-file' flag and a target path at the same time",
		},
		{
			// fail: scanio analyse --scanner semgrep /invalid/path/to/target
			name: "Invalid target path",
			options: RunOptionsAnalyse{
				Scanner: "semgrep",
			},
			args:          []string{"/invalid/path/to/target"},
			argsLenAtDash: -1,
			wantMode:      "",
			wantErr:       "the target path does not exist: /invalid/path/to/target",
		},
		// {
		// 	// scanio analyse --scanner semgrep --input-file /invalid/path/to/input.file
		// 	name: "Invalid input file",
		// 	options: RunOptionsAnalyse{
		// 		ScannerPluginName: "semgrep",
		// 		InputFile:         "/invalid/path/to/input.file",
		// 	},
		// 	args:          []string{},
		// 	argsLenAtDash: -1,
		// 	wantMode:      "",
		// 	wantErr:       "error parsing the input file /invalid/path/to/input.file",
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAnalyseArgs(&tt.options, tt.args, tt.argsLenAtDash)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				mode := determineMode(tt.args, tt.argsLenAtDash)
				assert.Equal(t, tt.wantMode, mode)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
