package template

import (
	"testing"
)

func TestLineAnchor(t *testing.T) {
	tests := []struct {
		name    string
		vcsType string
		line    int
		want    string
	}{
		{"github single line", "github", 10, "#L10"},
		{"github zero returns empty", "github", 0, ""},
		{"github negative returns empty", "github", -1, ""},
		{"gitlab single line", "gitlab", 5, "#L5"},
		{"bitbucket single line", "bitbucket", 15, "#15"},
		{"generic falls back to L-prefix", "generic", 7, "#L7"},
		{"empty vcsType falls back to L-prefix", "", 3, "#L3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lineAnchor(tt.vcsType, tt.line)
			if got != tt.want {
				t.Errorf("lineAnchor(%q, %d) = %q, want %q", tt.vcsType, tt.line, got, tt.want)
			}
		})
	}
}
