package fetcherutils

import (
	"testing"
)

func TestParsePRFetchMode(t *testing.T) {
	tests := []struct {
		input    string
		expected FetchMode
		wantErr  bool
	}{
		{"branch", PRBranchMode, false},
		{"ref", PRRefMode, false},
		{"commit", PRCommitMode, false},
		{"", PRBranchMode, false},
		{"invalid", PRBranchMode, true},
	}

	for _, tt := range tests {
		got, err := ParsePRFetchMode(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParsePRFetchMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if got != tt.expected {
			t.Errorf("ParsePRFetchMode(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
