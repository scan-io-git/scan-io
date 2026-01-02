package sarif

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"
)

type ScanioAnnotation string

const (
	GithubAnnotation    ScanioAnnotation = "> [!NOTE]\n> This issue was created and will be managed by scanio automation. Don't change body manually for proper processing, unless you know what you do."
	GitlabAnnotation                     = GenericAnnotation
	BitbucketAnnotation                  = GenericAnnotation
	GenericAnnotation   ScanioAnnotation = "⚠️ This issue was created and will be managed by scanio automation. Don't change body manually for proper processing, unless you know what you do."
)

func AnnotationByVCS(vcs string) string {
	switch strings.ToLower(vcs) {
	case "github":
		return string(GithubAnnotation)
	case "gitlab":
		return string(GitlabAnnotation)
	case "bitbucket":
		return string(BitbucketAnnotation)
	default:
		return string(GenericAnnotation)
	}
}

// LocationURLBuilder returns a permalink for a SARIF location when possible.
type LocationURLBuilder func(*sarif.Location) string

// NewLocationURLBuilder builds a provider-specific location URL strategy. When
// repository metadata is incomplete a noop builder is returned so callers can
// continue processing findings without URLs.
func NewLocationURLBuilder(meta *git.RepositoryMetadata, vcs string) (LocationURLBuilder, error) {
	noop := func(*sarif.Location) string { return "" }

	if meta == nil || meta.RepositoryFullName == nil || meta.CommitHash == nil {
		return noop, nil
	}

	vcsType := vcsurl.StringToVCSType(strings.ToLower(vcs))
	repoURL, err := vcsurl.ParseForVCSType(*meta.RepositoryFullName, vcsType)
	if err != nil {
		return noop, err
	}

	switch vcsType {
	case vcsurl.Bitbucket:
		return func(location *sarif.Location) string {
			return buildBitbucketLocationURL(location, repoURL, meta)
		}, nil
	default:
		return func(location *sarif.Location) string {
			return buildGenericLocationURL(location, repoURL, meta)
		}, nil
	}
}

func buildBitbucketLocationURL(location *sarif.Location, repoURL *vcsurl.VCSURL, meta *git.RepositoryMetadata) string {
	artifact := normalisedArtifactPath(location, meta)
	if artifact == "" {
		return ""
	}

	anchor := buildBitbucketAnchor(location)
	base := strings.TrimRight(repoURL.HTTPRepoLink, "/")
	return fmt.Sprintf("%s/browse/%s?at=%s%s", base, artifact, *meta.CommitHash, anchor)
}

func buildGenericLocationURL(location *sarif.Location, repoURL *vcsurl.VCSURL, meta *git.RepositoryMetadata) string {
	artifact := normalisedArtifactPath(location, meta)
	if artifact == "" {
		return ""
	}

	anchor := buildGenericAnchor(location)
	base := strings.TrimRight(repoURL.HTTPRepoLink, "/")
	return fmt.Sprintf("%s/blob/%s/%s%s", base, *meta.CommitHash, artifact, anchor)
}

func normalisedArtifactPath(location *sarif.Location, meta *git.RepositoryMetadata) string {
	if location == nil || location.PhysicalLocation == nil || location.PhysicalLocation.ArtifactLocation == nil {
		return ""
	}

	artifact := location.PhysicalLocation.ArtifactLocation
	if artifact.Properties == nil {
		artifact.Properties = make(map[string]interface{})
	}

	pathComponent := ""
	if val, ok := artifact.Properties["URI"].(string); ok && val != "" {
		pathComponent = filepath.ToSlash(val)
	} else {
		if artifact.URI == nil || *artifact.URI == "" {
			return ""
		}
		pathComponent = filepath.ToSlash(*artifact.URI)
		artifact.Properties["URI"] = pathComponent
	}

	if meta != nil {
		sub := strings.Trim(meta.Subfolder, "/\\")
		if sub != "" {
			pathComponent = path.Join(sub, pathComponent)
		}
	}
	return pathComponent
}

func buildBitbucketAnchor(location *sarif.Location) string {
	// url example: https://bitbucket.onprem.example/projects/<project_name>/repos/<repo_name>/browse/<path>/<vuln.file>?at=<commit_hash>#<line>
	if location == nil || location.PhysicalLocation == nil || location.PhysicalLocation.Region == nil {
		return ""
	}

	start := location.PhysicalLocation.Region.StartLine
	end := location.PhysicalLocation.Region.EndLine
	if start == nil || *start == 0 {
		return ""
	}

	anchor := "#" + strconv.Itoa(*start)
	if end != nil && *end != *start {
		anchor += "-" + strconv.Itoa(*end)
	}
	return anchor
}

// BuildBitbucketLocationURL constructs webURL for a report location for bitbucket.
// Deprecated: prefer NewLocationURLBuilder and the returned strategy.
func BuildBitbucketLocationURL(location *sarif.Location, url vcsurl.VCSURL, repoMetadata *git.RepositoryMetadata) string {
	return buildBitbucketLocationURL(location, &url, repoMetadata)
}

// BuildGenericLocationURL constructs webURL for a report location.
// Deprecated: prefer NewLocationURLBuilder and the returned strategy.
func BuildGenericLocationURL(location *sarif.Location, url vcsurl.VCSURL, repoMetadata *git.RepositoryMetadata) string {
	return buildGenericLocationURL(location, &url, repoMetadata)
}

func buildGenericAnchor(location *sarif.Location) string {
	if location == nil || location.PhysicalLocation == nil || location.PhysicalLocation.Region == nil {
		return ""
	}

	start := location.PhysicalLocation.Region.StartLine
	end := location.PhysicalLocation.Region.EndLine
	if start == nil || *start == 0 {
		return ""
	}

	anchor := "#L" + strconv.Itoa(*start)
	if end != nil && *end != *start {
		anchor += "-L" + strconv.Itoa(*end)
	}
	return anchor
}
