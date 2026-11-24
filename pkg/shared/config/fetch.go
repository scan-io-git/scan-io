package config

import (
	"fmt"
	"path/filepath"
)

// Fetch holds configuration for the fetch command's PR diff folders.
type Fetch struct {
	// DiffCopyPaths lists repository-root-relative files or folders copied in
	// full into the diff-lines and diff-files folders, changed or not. Unset
	// means DefaultDiffCopyPaths; an empty list copies nothing.
	DiffCopyPaths *[]string `yaml:"diff_copy_paths"`
}

// DefaultDiffCopyPaths are the config and ignore files that supported scanners
// load from the scan root on their own, so a scan of a diff folder still
// applies the repository's scanner settings. Each entry was checked against
// the scanner version installed by the Dockerfile.
//
// The defaults are resolved in the core process by ValidateFetchConfig, not in
// plugins: the config reaches plugins through gob, which decodes both an unset
// and an empty list as nil.
var DefaultDiffCopyPaths = []string{
	// Bandit reads .bandit from the target when run with -r, which the plugin
	// always passes: https://bandit.readthedocs.io/en/latest/config.html
	".bandit",
	// Betterleaks filesystem and git scans read <target>/.betterleaksignore:
	// https://github.com/betterleaks/betterleaks/blob/main/docs/scanning.md
	".betterleaksignore",
	// Semgrep reads .semgrepignore from the project root, which is the scan
	// root outside a git repository:
	// https://semgrep.dev/docs/semgrepignore-v2-reference
	".semgrepignore",
	// Trufflehog3 loads DEFAULT_CONFIG_FILE (".trufflehog3.yml") from the
	// target when --config is not set: trufflehog3/__init__.py, core.load_config.
	".trufflehog3.yml",
}

// CopyPaths returns the configured diff copy paths, or nil when none are set.
// It never falls back to DefaultDiffCopyPaths; ValidateFetchConfig does that.
func (f Fetch) CopyPaths() []string {
	if f.DiffCopyPaths == nil {
		return nil
	}
	return *f.DiffCopyPaths
}

// ValidateFetchConfig fills in DefaultDiffCopyPaths when the list is unset and
// rejects entries that are not paths inside the repository.
func ValidateFetchConfig(f *Fetch) error {
	if f == nil {
		return fmt.Errorf("fetch configuration is nil")
	}
	if f.DiffCopyPaths == nil {
		defaults := append([]string{}, DefaultDiffCopyPaths...)
		f.DiffCopyPaths = &defaults
		return nil
	}

	cleaned := make([]string, 0, len(*f.DiffCopyPaths))
	for _, entry := range *f.DiffCopyPaths {
		clean := filepath.Clean(entry)
		if !filepath.IsLocal(entry) || clean == "." {
			return fmt.Errorf("diff_copy_paths entry %q must be a path inside the repository, relative to its root", entry)
		}
		cleaned = append(cleaned, clean)
	}
	f.DiffCopyPaths = &cleaned
	return nil
}
