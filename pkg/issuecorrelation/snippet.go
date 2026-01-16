package issuecorrelation

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
)

// ComputeSnippetHash reads the snippet (single line or range) from a local filesystem path
// and returns its SHA256 hex string. Returns empty string on any error or if inputs are invalid.
func ComputeSnippetHash(localPath string, line, endLine int) string {
	if strings.TrimSpace(localPath) == "" || line <= 0 {
		return ""
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	start := line
	end := line
	if endLine > line {
		end = endLine
	}
	// Validate bounds (1-based line numbers)
	if start < 1 || start > len(lines) {
		return ""
	}
	if end > len(lines) {
		end = len(lines)
	}
	if end < start {
		return ""
	}
	snippet := strings.Join(lines[start-1:end], "\n")
	sum := sha256.Sum256([]byte(snippet))
	return fmt.Sprintf("%x", sum[:])
}
