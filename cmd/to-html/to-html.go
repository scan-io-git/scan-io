package tohtml

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"

	scaniosarif "github.com/scan-io-git/scan-io/internal/sarif"
	scaniotemplate "github.com/scan-io-git/scan-io/internal/template"
)

const (
	defaultHtmlTemplateHome = "templates/tohtml/"
	defaultHtmlTemplateName = "report.html"
)

type ToHTMLOptions struct {
	TempatesPath   string `json:"tempates_path,omitempty"`
	Title          string `json:"title,omitempty"`
	OutputFile     string `json:"output_file,omitempty"`
	Input          string `json:"input,omitempty"`
	SourceFolder   string `json:"source_folder,omitempty"`
	VCS            string `json:"vcs,omitempty"`
	PullRequest    string `json:"pull_request,omitempty"`
	NoSuppressions bool   `json:"nosuppressions,omitempty"`
	NoCSP          bool   `json:"no_csp,omitempty"`
	Required       string `json:"required,omitempty"`
}

type VCSURLInfo struct {
	VCSType       string
	Hostname      string
	Namespace     string
	Repository    string
	Branch        string
	PullRequestId string
	HTTPRepoLink  string
	SSHRepoLink   string
}

type cspData struct {
	Enabled bool
	Nonce   string
}

type ReportMetadata struct {
	git.RepositoryMetadata
	scaniosarif.ToolMetadata
	Title        string
	Time         time.Time
	SourceFolder string
	SeverityInfo    map[string]int
	SuppressionInfo map[string]int
	RequiredEnabled bool
	RequiredInfo    map[string]int
	WebURL       string
	BranchURL    string
	CommitURL    string
	PRURL        string
	VCSURL       *VCSURLInfo
}

// Global variables for configuration and command arguments
var (
	AppConfig         *config.Config
	logger            hclog.Logger
	allToHTMLOptions  ToHTMLOptions
	execExampleToHTML = `  # Generate html report for semgrep sarif output
  scanio to-html --input /tmp/juice-shop/semgrep.sarif --output /tmp/juice-shop/semgrep.html --source /tmp/juice-shop

  # Generate html report for semgrep sarif output, use bitbucket specific hyperlink URL builder
  scanio to-html --input /tmp/juice-shop/semgrep.sarif --output /tmp/juice-shop/semgrep.html --source /tmp/juice-shop --vcs bitbucket

  # Use custom templates path for html report generation
  scanio to-html -i /tmp/juice-shop/semgrep_results.sarif -o /tmp/juice-shop/semgrep_results.html -s /tmp/juice-shop/ -t ./templates/tohtml

  # Use no-supressions to skip results with supressions sarif property
  scanio to-html -i /tmp/juice-shop/semgrep_results.sarif -o /tmp/juice-shop/semgrep_results.html -s /tmp/juice-shop/ -t ./templates/tohtml --no-supressions`
)

func vcsTypeToString(t vcsurl.VCSType) string {
	switch t {
	case vcsurl.Github:
		return "github"
	case vcsurl.Gitlab:
		return "gitlab"
	case vcsurl.Bitbucket:
		return "bitbucket"
	case vcsurl.GenericVCS:
		return "generic"
	default:
		return ""
	}
}

// vcsHostMap converts the config VCSHosts string map to the vcsurl.VCSType map required by
// ParseOptions. Entries with unrecognised type strings are skipped.
func vcsHostMap(cfg *config.Config) map[string]vcsurl.VCSType {
	if cfg == nil || len(cfg.VCSHosts) == 0 {
		return nil
	}
	m := make(map[string]vcsurl.VCSType, len(cfg.VCSHosts))
	for host, typeName := range cfg.VCSHosts {
		if t := vcsurl.StringToVCSType(typeName); t != vcsurl.UnknownVCS {
			m[host] = t
		}
	}
	return m
}

// locationRegion extracts start and end line numbers from a SARIF location, returning zeros
// when region or line pointers are nil.
func locationRegion(loc *sarif.Location) (startLine, endLine int) {
	if loc.PhysicalLocation == nil || loc.PhysicalLocation.Region == nil {
		return 0, 0
	}
	r := loc.PhysicalLocation.Region
	if r.StartLine != nil {
		startLine = *r.StartLine
	}
	if r.EndLine != nil {
		endLine = *r.EndLine
	}
	return startLine, endLine
}

// Init initializes the global configuration variable.
func Init(cfg *config.Config, l hclog.Logger) {
	AppConfig = cfg
	logger = l
}

// ToHtmlCmd represents the toHtml command
var ToHtmlCmd = &cobra.Command{
	Use:     "to-html -i /path/to/input/report.sarif -o /path/to/output/report.html -s /path/to/source/folder",
	Short:   "Generate HTML formatted report",
	Example: execExampleToHTML,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("to-html called")

		sarifReport, err := scaniosarif.ReadReport(allToHTMLOptions.Input, logger, allToHTMLOptions.SourceFolder, allToHTMLOptions.NoSuppressions)
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
		}

		repositoryMetadata, err := git.CollectRepositoryMetadata(allToHTMLOptions.SourceFolder)
		if err != nil {
			logger.Warn("can't collect repository metadata", "reason", err)
		} else {
			branch := ""
			if repositoryMetadata.BranchName != nil {
				branch = *repositoryMetadata.BranchName
			}
			commit := ""
			if repositoryMetadata.CommitHash != nil {
				commit = *repositoryMetadata.CommitHash
			}
			fullName := ""
			if repositoryMetadata.RepositoryFullName != nil {
				fullName = *repositoryMetadata.RepositoryFullName
			}
			logger.Debug(
				"repositoryMetadata",
				"BranchName", branch,
				"CommitHash", commit,
				"RepositoryFullName", fullName,
				"Subfolder", repositoryMetadata.Subfolder,
				"RepoRootFolder", repositoryMetadata.RepoRootFolder,
			)
		}

		var parsedURL *vcsurl.VCSURL
		if repositoryMetadata.RepositoryFullName != nil {
			vcsType := vcsurl.StringToVCSType(allToHTMLOptions.VCS)
			if vcsType == vcsurl.UnknownVCS {
				// No --vcs flag: run full detection chain including config host map.
				parsedURL, err = vcsurl.ParseWithOptions(*repositoryMetadata.RepositoryFullName, vcsurl.ParseOptions{
					HostMap: vcsHostMap(AppConfig),
				})
			} else {
				// Explicit --vcs value: bypass detection.
				parsedURL, err = vcsurl.ParseForVCSType(*repositoryMetadata.RepositoryFullName, vcsType)
			}
			if err != nil {
				return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
			}
		}

		if parsedURL != nil {
			if prID := resolvePullRequestID(allToHTMLOptions.PullRequest, parsedURL.VCSType); prID != "" {
				parsedURL.PullRequestId = prID
			}
		}

		var commitHash string
		if repositoryMetadata.CommitHash != nil {
			commitHash = *repositoryMetadata.CommitHash
		}

		// locationURLCallback produces the file-at-commit permalink for a SARIF location.
		// It reads the URI property set earlier by EnrichResultsLocationProperty.
		locationURLCallback := func(loc *sarif.Location) string {
			if parsedURL == nil {
				return ""
			}
			uri, _ := loc.PhysicalLocation.ArtifactLocation.Properties["URI"].(string)
			startLine, endLine := locationRegion(loc)
			return parsedURL.FilePermalink(commitHash, uri, startLine, endLine)
		}

		// prDiffURLCallback produces a PR-diff deep link when a pull-request ID is present.
		prDiffURLCallback := func(loc *sarif.Location) string {
			if parsedURL == nil || parsedURL.PullRequestId == "" {
				return ""
			}
			uri, _ := loc.PhysicalLocation.ArtifactLocation.Properties["URI"].(string)
			startLine, _ := locationRegion(loc)
			return parsedURL.PRDiffLink(uri, startLine)
		}

		sarifReport.EnrichResultsTitleProperty()
		sarifReport.EnrichResultsCodeFlowProperty(locationURLCallback)
		sarifReport.RemoveDataflowDuplicates()
		sarifReport.EnrichResultsLevelProperty()
		sarifReport.EnrichResultsCategoryProperty()
		sarifReport.EnrichResultsConfidenceProperty()
		sarifReport.EnrichResultsMetadataProperty()
		sarifReport.EnrichResultsLocationURIProperty(locationURLCallback, prDiffURLCallback)
		sarifReport.EnrichResultsSuppressionProperty()

		requiredPolicy, requiredEnabled := parseRequiredPolicy(allToHTMLOptions.Required)
		if requiredEnabled {
			sarifReport.EnrichResultsRequiredProperty(requiredPolicy)
			sarifReport.SortResultsByRequiredThenSeverity()
		} else {
			sarifReport.SortResultsBySeverity()
		}

		toolMetadata, err := sarifReport.ExtractToolNameAndVersion()
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
		}
		logger.Debug("toolMetadata", "Name", toolMetadata.Name, "Version", toolMetadata.Version)

		severityInfo := sarifReport.CollectSeverityInfo()
		suppressionInfo := sarifReport.CollectSuppressionInfo()
		requiredInfo := map[string]int{"required": 0, "recommended": 0}
		if requiredEnabled {
			requiredInfo = sarifReport.CollectRequiredInfo()
		}

		metadataSourceFolder := allToHTMLOptions.SourceFolder
		if config.IsCI(AppConfig) {
			metadataSourceFolder = ""
		}

		metadata := &ReportMetadata{
			RepositoryMetadata: *repositoryMetadata,
			ToolMetadata:       *toolMetadata,
			Title:              allToHTMLOptions.Title,
			Time:               time.Now().UTC(),
			SourceFolder:       metadataSourceFolder,
			SeverityInfo:       severityInfo,
			SuppressionInfo:    suppressionInfo,
			RequiredEnabled:    requiredEnabled,
			RequiredInfo:       requiredInfo,
		}
		if parsedURL != nil {
			metadata.WebURL = parsedURL.HTTPRepoLink
			metadata.PRURL = parsedURL.PRURL()
			metadata.VCSURL = &VCSURLInfo{
				VCSType:       vcsTypeToString(parsedURL.VCSType),
				Hostname:      parsedURL.ParsedURL.Hostname(),
				Namespace:     parsedURL.Namespace,
				Repository:    parsedURL.Repository,
				Branch:        parsedURL.Branch,
				PullRequestId: parsedURL.PullRequestId,
				HTTPRepoLink:  parsedURL.HTTPRepoLink,
				SSHRepoLink:   parsedURL.SSHRepoLink,
			}
			if repositoryMetadata.BranchName != nil {
				metadata.BranchURL = parsedURL.BranchURL(*repositoryMetadata.BranchName)
			}
			if repositoryMetadata.CommitHash != nil {
				metadata.CommitURL = parsedURL.CommitURL(*repositoryMetadata.CommitHash)
			}
		}

		logger.Debug("metadata", "metadata", *metadata)

		if allToHTMLOptions.TempatesPath == "" {
			allToHTMLOptions.TempatesPath = filepath.Join(config.GetScanioHome(AppConfig), defaultHtmlTemplateHome)
		}

		templateFullPath, _, err := files.DetermineFileFullPath(allToHTMLOptions.TempatesPath, defaultHtmlTemplateName)
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, fmt.Errorf("failed to determine html template path for %q: %w", allToHTMLOptions.TempatesPath, err), 1)
		}

		tmpl, err := scaniotemplate.NewTemplate(templateFullPath)
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
		}

		cspNonce := ""
		if !allToHTMLOptions.NoCSP {
			raw := make([]byte, 16)
			if _, err := rand.Read(raw); err != nil {
				return errors.NewCommandError(allToHTMLOptions, nil, fmt.Errorf("failed to generate CSP nonce: %w", err), 1)
			}
			cspNonce = base64.RawURLEncoding.EncodeToString(raw)
		}

		data := struct {
			Metadata *ReportMetadata
			Report   *scaniosarif.Report
			CSP      cspData
		}{
			Metadata: metadata,
			Report:   sarifReport,
			CSP: cspData{
				Enabled: !allToHTMLOptions.NoCSP,
				Nonce:   cspNonce,
			},
		}

		file, err := os.Create(allToHTMLOptions.OutputFile)
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
		}
		defer file.Close()

		err = tmpl.Execute(file, data)
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
		}

		logger.Info("html report saved to file", "path", allToHTMLOptions.OutputFile)

		return nil
	},
}

func init() {
	ToHtmlCmd.Flags().StringVarP(&allToHTMLOptions.TempatesPath, "templates-path", "t", "", "Path to folder with templates")
	ToHtmlCmd.Flags().StringVar(&allToHTMLOptions.Title, "title", "Scanio Report", "Title for generated html file")
	ToHtmlCmd.Flags().StringVarP(&allToHTMLOptions.Input, "input", "i", "", "Input file with sarif report")
	ToHtmlCmd.Flags().StringVarP(&allToHTMLOptions.OutputFile, "output", "o", "scanio-report.html", "output file")
	ToHtmlCmd.Flags().StringVarP(&allToHTMLOptions.SourceFolder, "source", "s", "", "Source folder")
	ToHtmlCmd.Flags().StringVar(&allToHTMLOptions.VCS, "vcs", "", "VCS type override (github, gitlab, bitbucket, generic); leave empty to auto-detect")
	ToHtmlCmd.Flags().StringVar(&allToHTMLOptions.PullRequest, "pull-request", "", "Pull request ID; enables PR-aware links in the report. Falls back to CI env vars (GITHUB_REF, CI_MERGE_REQUEST_IID, BITBUCKET_PR_ID) when omitted.")
	ToHtmlCmd.Flags().BoolVarP(&allToHTMLOptions.NoSuppressions, "no-supressions", "", false, "Enable removing results with suppressions properties")
	ToHtmlCmd.Flags().BoolVar(&allToHTMLOptions.NoCSP, "no-csp", false, "Disable Content-Security-Policy meta tag in generated report")
	ToHtmlCmd.Flags().StringVar(&allToHTMLOptions.Required, "required", "", "Enable Required/Recommended classification. Comma list of blocker severities with optional per-severity confidence threshold, e.g. \"critical:0.50,high\". Falls back to SCANIO_BLOCKER_SEVERITIES and SCANIO_CONFIDENCE_THRESHOLD_<SEV> env vars when omitted.")
}
