package tohtml

import (
	"strings"

	"github.com/scan-io-git/scan-io/internal/git"
	scaniosarif "github.com/scan-io-git/scan-io/internal/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"
)

// provenance is the repository identity a SARIF run declares for itself.
type provenance struct {
	repo   string
	commit *string
	branch *string
}

// trimmedPtr returns a pointer to the trimmed value, or nil when it is absent or blank.
func trimmedPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

// firstProvenance returns the first versionControlProvenance entry, across all
// inputs and runs, that has a repository URI.
func firstProvenance(reports []*scaniosarif.Report) (provenance, bool) {
	for _, report := range reports {
		for _, run := range report.Runs {
			for _, vc := range run.VersionControlProvenance {
				if vc == nil {
					continue
				}
				if repo := trimmedPtr(vc.RepositoryURI); repo != nil {
					return provenance{repo: *repo, commit: trimmedPtr(vc.RevisionID), branch: trimmedPtr(vc.Branch)}, true
				}
			}
		}
	}
	return provenance{}, false
}

// applyProvenance fills repository, commit and branch from the SARIF's own
// versionControlProvenance and returns the parsed repository URL. Callers must
// invoke it only when git supplied no repository URL, because git is the
// authority otherwise. Git's metadata is replaced only once the provenance URL
// parses, and then as a whole: commit and branch are never mixed with local git
// values that describe a different source. A missing or unusable entry leaves
// the metadata untouched and returns nil, which costs the links, not the report.
func applyProvenance(md *git.RepositoryMetadata, reports []*scaniosarif.Report) *vcsurl.VCSURL {
	p, ok := firstProvenance(reports)
	if !ok {
		return nil
	}
	parsed, err := parseRepositoryURL(p.repo)
	if err != nil {
		logger.Warn("ignoring unparsable repositoryUri in versionControlProvenance", "reason", err)
		return nil
	}
	md.RepositoryFullName, md.CommitHash, md.BranchName = &p.repo, p.commit, p.branch
	logger.Info("using versionControlProvenance from the sarif input for repository links")
	return parsed
}

// parseRepositoryURL parses a repository URL honoring the --vcs override and the
// configured host map.
func parseRepositoryURL(raw string) (*vcsurl.VCSURL, error) {
	vcsType := vcsurl.StringToVCSType(allToHTMLOptions.VCS)
	if vcsType == vcsurl.UnknownVCS {
		// No --vcs flag: run full detection chain including config host map.
		return vcsurl.ParseWithOptions(raw, vcsurl.ParseOptions{HostMap: vcsHostMap(AppConfig)})
	}
	// Explicit --vcs value: bypass detection.
	return vcsurl.ParseForVCSType(raw, vcsType)
}
