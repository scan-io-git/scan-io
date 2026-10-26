package issuecorrelation

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

// ComputeSnippetHash reads the snippet (single line or range) from path and returns its
// SHA256 hex string. path is relative to root or absolute inside it; symlinks and paths
// that lead outside root are refused. Returns empty string on any error or if inputs are invalid.
func ComputeSnippetHash(root, path string, line, endLine int) string {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(path) == "" || line <= 0 {
		return ""
	}
	f, err := files.OpenInRootPath(root, path)
	if err != nil {
		return ""
	}
	defer f.Close()
	data, err := io.ReadAll(f)
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
