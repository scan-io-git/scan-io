package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"

	scaniosarif "github.com/scan-io-git/scan-io/internal/sarif"
	scaniotemplate "github.com/scan-io-git/scan-io/internal/template"
)

type ToHTMLOptions struct {
	TempatesPath string `json:"tempates_path,omitempty"`
	Title        string `json:"title,omitempty"`
	OutputFile   string `json:"output_file,omitempty"`
	Input        string `json:"input,omitempty"`
	SourceFolder string `json:"source_folder,omitempty"`
	VCS          string `json:"vcs,omitempty"`
}

var allToHTMLOptions ToHTMLOptions

type ReportMetadata struct {
	git.RepositoryMetadata
	scaniosarif.ToolMetadata
	Title        string
	Time         time.Time
	SourceFolder string
	SeverityInfo map[string]int
	WebURL       string
	BranchURL    string
	CommitURL    string
}

var execExampleToHTML = `  # Generate html report for semgrep sarif output
  scanio to-html --input /tmp/juice-shop/semgrep.sarif --output /tmp/juice-shop/semgrep.html --source /tmp/juice-shop

  # Generate html report for semgrep sarif output, use bitbucket specific hyperlink URL builder
  scanio to-html --input /tmp/juice-shop/semgrep.sarif --output /tmp/juice-shop/semgrep.html --source /tmp/juice-shop --vcs bitbucket

  # Use custom templates path for html report generation
  scanio to-html -i /tmp/juice-shop/semgrep_results.sarif -o /tmp/juice-shop/semgrep_results.html -s /tmp/juice-shop/ -t ./templates/tohtml`

// this function will implement vcs specific logic to generate web URL to branch or commit + special case for onprem BB
func buildWebURLToRef(url *vcsurl.VCSURL, refName, refType string) string {
	var midder string
	switch refType {
	case "commit":
		midder = "commits"
	case "branch":
		if url.VCSType == vcsurl.Bitbucket {
			// Special case for Bitbucket branch
			return filepath.Join(url.HTTPRepoLink, "browse?at=refs%2Fheads%2F"+refName)
		}
		midder = "tree"
	}
	return filepath.Join(url.HTTPRepoLink, midder, refName)
}

// buildGenericLocationURL constructs webURL for a report location
func buildGenericLocationURL(location *sarif.Location, url vcsurl.VCSURL, repoMetadata *git.RepositoryMetadata) string {
	// verify that location.PhysicalLocation.ArtifactLocation.Properties["URI"] is not nil
	if location.PhysicalLocation.ArtifactLocation.Properties["URI"] == nil {
		return ""
	}
	locationWebURL := filepath.Join(url.HTTPRepoLink, "blob", *repoMetadata.CommitHash, location.PhysicalLocation.ArtifactLocation.Properties["URI"].(string))
	if location.PhysicalLocation.Region.StartLine != nil {
		locationWebURL += "#L" + strconv.Itoa(*location.PhysicalLocation.Region.StartLine)
	}
	if location.PhysicalLocation.Region.EndLine != nil && *location.PhysicalLocation.Region.EndLine != *location.PhysicalLocation.Region.StartLine {
		locationWebURL += "-L" + strconv.Itoa(*location.PhysicalLocation.Region.EndLine)
	}
	return locationWebURL
}

// buildBitbucketLocationURL constructs webURL for a report location for bitbucket
func buildBitbucketLocationURL(location *sarif.Location, url vcsurl.VCSURL, repoMetadata *git.RepositoryMetadata) string {
	// url example: https://bitbucket.onprem.example/projects/<project_name>/repos/<repo_name>/browse/<path>/<vuln.file>?at=<commit_hash>#<line>
	// verify that location.PhysicalLocation.ArtifactLocation.Properties["URI"] is not nil
	if location.PhysicalLocation.ArtifactLocation.Properties["URI"] == nil {
		return ""
	}
	locationWebURL := filepath.Join(url.HTTPRepoLink, "browse", location.PhysicalLocation.ArtifactLocation.Properties["URI"].(string))
	locationWebURL += "?at=" + *repoMetadata.CommitHash
	if location.PhysicalLocation.Region.StartLine != nil {
		locationWebURL += "#" + strconv.Itoa(*location.PhysicalLocation.Region.StartLine)
	}
	if location.PhysicalLocation.Region.EndLine != nil && *location.PhysicalLocation.Region.EndLine != *location.PhysicalLocation.Region.StartLine {
		locationWebURL += "-" + strconv.Itoa(*location.PhysicalLocation.Region.EndLine)
	}
	return locationWebURL
}

// toHtmlCmd represents the toHtml command
var toHtmlCmd = &cobra.Command{
	Use:     "to-html -i /path/to/input/report.sarif -o /path/to/output/report.html -s /path/to/source/folder",
	Short:   "Generate HTML formatted report",
	Example: execExampleToHTML,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logger.NewLogger(AppConfig, "core")
		logger.Info("to-html called")

		// read sarif report from file
		sarifReport, err := scaniosarif.ReadReport(allToHTMLOptions.Input, logger, allToHTMLOptions.SourceFolder)
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
		}

		// collect metadata for the report template
		repositoryMetadata, err := git.CollectRepositoryMetadata(allToHTMLOptions.SourceFolder)
		if err != nil {
			logger.Warn("can't collect repository metadata", "reason", err)
		} else {
			logger.Debug("repositoryMetadata", "BranchName", *repositoryMetadata.BranchName, "CommitHash", *repositoryMetadata.CommitHash, "RepositoryFullName", *repositoryMetadata.RepositoryFullName, "Subfolder", repositoryMetadata.Subfolder, "RepoRootFolder", repositoryMetadata.RepoRootFolder)
		}

		var url *vcsurl.VCSURL
		vcsType := vcsurl.GenericVCS
		if repositoryMetadata.RepositoryFullName != nil {
			vcsType = vcsurl.StringToVCSType(allToHTMLOptions.VCS)
			url, err = vcsurl.ParseForVCSType(*repositoryMetadata.RepositoryFullName, vcsType)
			if err != nil {
				return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
			}
		}

		// a callback function to generate web url for location
		// we need it because neither sarif nor git modules know anything about vcs web URL structures.
		// so we should implement vcs scpecific logic here
		// for beginning I started with generic/github implementation
		locationWebURLCallback := func(location *sarif.Location) string {
			if url == nil {
				return ""
			}
			if vcsType == vcsurl.Bitbucket {
				return buildBitbucketLocationURL(location, *url, repositoryMetadata)
			}
			return buildGenericLocationURL(location, *url, repositoryMetadata)
		}

		// enrich sarif report with additional properties and remove duplicates from dataflow results
		sarifReport.EnrichResultsTitleProperty()
		sarifReport.EnrichResultsCodeFlowProperty(locationWebURLCallback)
		sarifReport.EnrichResultsLevelProperty()
		sarifReport.EnrichResultsLocationURIProperty(locationWebURLCallback)
		sarifReport.SortResultsByLevel()
		sarifReport.RemoveDataflowDuplicates()

		toolMetadata, err := sarifReport.ExtractToolNameAndVersion()
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
		}
		logger.Debug("toolMetadata", "Name", toolMetadata.Name, "Version", toolMetadata.Version)

		severityInfo := sarifReport.CollectSeverityInfo()

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
		}
		if url != nil {
			metadata.WebURL = url.HTTPRepoLink
		}
		if repositoryMetadata.BranchName != nil {
			metadata.BranchURL = buildWebURLToRef(url, *repositoryMetadata.BranchName, "branch")
		}
		if repositoryMetadata.CommitHash != nil {
			metadata.CommitURL = buildWebURLToRef(url, *repositoryMetadata.CommitHash, "commit")
		}

		logger.Debug("metadata", "metadata", *metadata)

		// parse html template and generate report file with metadata
		if allToHTMLOptions.TempatesPath == "" {
			allToHTMLOptions.TempatesPath = filepath.Join(config.GetScanioHome(AppConfig), "templates/tohtml/")
		}

		templateFile, err := files.ExpandPath(filepath.Join(allToHTMLOptions.TempatesPath, "report.html"))
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
		}

		tmpl, err := scaniotemplate.NewTemplate(templateFile)
		if err != nil {
			return errors.NewCommandError(allToHTMLOptions, nil, err, 1)
		}

		data := struct {
			Metadata *ReportMetadata
			Report   *scaniosarif.Report
		}{
			Metadata: metadata,
			Report:   sarifReport,
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
	rootCmd.AddCommand(toHtmlCmd)

	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.TempatesPath, "templates-path", "t", "", "path to folder with templates")
	toHtmlCmd.Flags().StringVar(&allToHTMLOptions.Title, "title", "Scanio Report", "title for generated html file")
	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.Input, "input", "i", "", "input file with sarif report")
	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.OutputFile, "output", "o", "scanio-report.html", "outoput file")
	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.SourceFolder, "source", "s", "", "source folder")
	toHtmlCmd.Flags().StringVar(&allToHTMLOptions.VCS, "vcs", "generic", "vcs type")
}
