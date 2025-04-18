package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetermineFileFullPath(t *testing.T) {
	type testCase struct {
		name         string
		inputPath    string
		nameTemplate string
		expectFile   string
		expectFolder string
		setup        func(t *testing.T) (inputPath, expectFile, expectFolder string)
	}

	tmpDir := t.TempDir()

	tests := []testCase{
		{
			name:         "Directory path with name template",
			inputPath:    tmpDir,
			nameTemplate: "output.json",
			expectFile:   filepath.Join(tmpDir, "output.json"),
			expectFolder: tmpDir,
		},
		{
			name:         "File path with extension",
			inputPath:    filepath.Join(tmpDir, "data.json"),
			nameTemplate: "ignored.txt",
			expectFile:   filepath.Join(tmpDir, "data.json"),
			expectFolder: tmpDir,
			setup: func(t *testing.T) (string, string, string) {
				f := filepath.Join(tmpDir, "data.json")
				_ = os.WriteFile(f, []byte("test"), 0644)
				return f, f, tmpDir
			},
		},
		{
			name:         "Path with no extension, treat as folder",
			inputPath:    filepath.Join(tmpDir, "output_folder"),
			nameTemplate: "report.log",
			expectFile:   filepath.Join(tmpDir, "output_folder", "report.log"),
			expectFolder: filepath.Join(tmpDir, "output_folder"),
		},
		{
			name:         "Non-existent file with extension",
			inputPath:    filepath.Join(tmpDir, "nonexistent.yaml"),
			nameTemplate: "ignored.txt",
			expectFile:   filepath.Join(tmpDir, "nonexistent.yaml"),
			expectFolder: tmpDir,
		},
		{
			name:         "Non-existent folder",
			inputPath:    filepath.Join(tmpDir, "missing_folder"),
			nameTemplate: "result.json",
			expectFile:   filepath.Join(tmpDir, "missing_folder", "result.json"),
			expectFolder: filepath.Join(tmpDir, "missing_folder"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPath := tt.inputPath
			expectFile := tt.expectFile
			expectFolder := tt.expectFolder

			if tt.setup != nil {
				actualPath, expectFile, expectFolder = tt.setup(t)
			}

			filePath, folderPath, err := DetermineFileFullPath(actualPath, tt.nameTemplate)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if filePath != expectFile {
				t.Errorf("Expected file path %s, got %s", expectFile, filePath)
			}
			if folderPath != expectFolder {
				t.Errorf("Expected folder path %s, got %s", expectFolder, folderPath)
			}
		})
	}
}
