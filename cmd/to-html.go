package cmd

import (
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/git"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"

	scaniosarif "github.com/scan-io-git/scan-io/internal/sarif"
	scaniotemplate "github.com/scan-io-git/scan-io/internal/template"
)

type ToHTMLOptions struct {
	TempatesPath string
	Title        string
	OutputFile   string
	Input        string
	SourceFolder string
}

var allToHTMLOptions ToHTMLOptions

type ReportMetadata struct {
	git.RepositoryMetadata
	scaniosarif.ToolMetadata
	Title        string
	Time         time.Time
	SourceFolder string
	SeverityInfo map[string]int
}

var execExampleToHTML = `  # Generate html report for semgrep sarif output
  scanio to-html --input /tmp/juice-shop/semgrep.sarif --output /tmp/juice-shop/semgrep.html --source /tmp/juice-shop`

func gitURLtoWebURL(gitURL string) string {
	u, err := vcsurl.Parse(gitURL)
	if err != nil {
		return gitURL
	}
	return u.HTTPRepoLink
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
			return err
		}

		// enrich sarif report with additional properties and remove duplicates from dataflow results
		sarifReport.EnrichResultsProperties()
		sarifReport.SortResultsByLevel()
		sarifReport.RemoveDataflowDuplicates()

		// collect metadata for the report template
		repositoryMetadata, err := git.CollectRepositoryMetadata(allToHTMLOptions.SourceFolder)
		if err != nil {
			logger.Debug("can't collect repository metadata", "err", err)
		}
		logger.Debug("repositoryMetadata", "BranchName", *repositoryMetadata.BranchName, "CommitHash", *repositoryMetadata.CommitHash, "RepositoryFullName", *repositoryMetadata.RepositoryFullName, "Subfolder", repositoryMetadata.Subfolder, "RepoRootFolder", repositoryMetadata.RepoRootFolder)

		toolMetadata, err := sarifReport.ExtractToolNameAndVersion()
		if err != nil {
			return err
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
		logger.Debug("metadata", "metadata", *metadata)

		// parse html template and generate report file with metadata
		if allToHTMLOptions.TempatesPath == "" {
			allToHTMLOptions.TempatesPath = filepath.Join(config.GetScanioHome(AppConfig), "templates/tohtml/")
		}

		templateFile, err := files.ExpandPath(filepath.Join(allToHTMLOptions.TempatesPath, "report.html"))
		if err != nil {
			return err
		}

		tmpl, err := scaniotemplate.NewTemplate(templateFile, scaniotemplate.WithFuncs(template.FuncMap{"gitURLtoWebURL": gitURLtoWebURL}))
		if err != nil {
			return err
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
			return err
		}
		defer file.Close()

		err = tmpl.Execute(file, data)
		if err != nil {
			return err
		}

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
}
