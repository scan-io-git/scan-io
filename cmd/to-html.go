package cmd

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/spf13/cobra"

	scaniosarif "github.com/scan-io-git/scan-io/internal/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

type ToHTMLOptions struct {
	TempatesPath string
	Title        string
	OutputFile   string
	Input        string
	SourceFolder string
}

var allToHTMLOptions ToHTMLOptions

// add adds two integers and returns the result.
// helper function for html template
func add(a, b int) int {
	return a + b
}

// generateSequence generates a slice of integers from 1 to n.
// helper function for html template
func generateSequence(n int) []int {
	var sequence []int
	for i := 1; i <= n; i++ {
		sequence = append(sequence, i)
	}
	return sequence
}

// findGitRepositoryPath function finds a git repository path for a given source folder
func findGitRepositoryPath(sourceFolder string) (string, error) {
	if sourceFolder == "" {
		return "", fmt.Errorf("source folder is not set")
	}

	// check if source folder is a subfolder of a git repository
	for {
		_, err := git.PlainOpen(sourceFolder)
		if err == nil {
			return sourceFolder, nil
		}

		// move up one level
		sourceFolder = filepath.Dir(sourceFolder)

		// check if reached the root folder
		if sourceFolder == filepath.Dir(sourceFolder) {
			break
		}
	}

	return "", fmt.Errorf("source folder is not a git repository")
}

// struct with repository metadata
type RepositoryMetadata struct {
	BranchName         *string
	CommitHash         *string
	RepositoryFullName *string
	Subfolder          string
	RepoRootFolder     string
}

type ReportMetadata struct {
	RepositoryMetadata
	scaniosarif.ToolMetadata
	Title        string
	Time         time.Time
	SourceFolder string
	SeverityInfo map[string]int
}

// collectRepositoryMetadata function collects repository metadata
// that includes branch name, commit hash, repository full name, subfolder and repository root folder
func collectRepositoryMetadata() (*RepositoryMetadata, error) {
	defaultRepositoryMetadata := &RepositoryMetadata{
		RepoRootFolder: allToHTMLOptions.SourceFolder,
		Subfolder:      "",
	}

	if allToHTMLOptions.SourceFolder == "" {
		return defaultRepositoryMetadata, fmt.Errorf("source folder is not set")
	}

	repoRootFolder, err := findGitRepositoryPath(allToHTMLOptions.SourceFolder)
	if err != nil {
		return defaultRepositoryMetadata, err
	}

	repo, err := git.PlainOpen(repoRootFolder)
	if err != nil {
		return defaultRepositoryMetadata, fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return defaultRepositoryMetadata, fmt.Errorf("failed to get HEAD: %w", err)
	}
	branchName := head.Name().Short()
	commitHash := head.Hash().String()

	remote, err := repo.Remote("origin")
	if err != nil {
		return defaultRepositoryMetadata, fmt.Errorf("failed to get remote: %w", err)
	}

	repositoryFullName := strings.TrimSuffix(remote.Config().URLs[0], ".git")

	return &RepositoryMetadata{
		BranchName:         &branchName,
		CommitHash:         &commitHash,
		RepositoryFullName: &repositoryFullName,
		Subfolder:          strings.TrimPrefix(allToHTMLOptions.SourceFolder, repoRootFolder),
		RepoRootFolder:     repoRootFolder,
	}, nil
}

// ordinalDate returns a string with the ordinal number of the day
// helper function for html template
func ordinalDate(day int) string {
	suffix := "th"
	switch day {
	case 1, 21, 31:
		suffix = "st"
	case 2, 22:
		suffix = "nd"
	case 3, 23:
		suffix = "rd"
	}
	return fmt.Sprintf("%d%s", day, suffix)
}

// formatDateTime formats a time.Time object into the specified string format.
// helper function for html template
func formatDateTime(t time.Time) string {
	day := ordinalDate(t.Day())
	return fmt.Sprintf("%s %s %d %d:%02d:%02d %s", day, t.Month(), t.Year(), t.Hour()%12, t.Minute(), t.Second(), t.Format("pm"))
}

// sortResultsByLevel function sorts sarif results by level
func sortResultsByLevel(results []*sarif.Result) {
	// sort results by level
	// order: error, warning, note, none
	levelOrder := map[string]int{
		"error":   0,
		"warning": 1,
		"note":    2,
		"none":    3,
		"unknown": 4,
	}

	// sort results by level
	// order: error, warning, note, none, unknown
	sort.Slice(results, func(i, j int) bool {
		return levelOrder[results[i].Properties["Level"].(string)] < levelOrder[results[j].Properties["Level"].(string)]
	})
}

var execExampleToHTML = `  # Generate html report for semgrep sarif output
  scanio to-html --input /tmp/juice-shop/semgrep.sarif --output /tmp/juice-shop/semgrep.html --source /tmp/juice-shop`

// toHtmlCmd represents the toHtml command
var toHtmlCmd = &cobra.Command{
	Use:     "to-html -i /path/to/input/report.sarif -o /path/to/output/report.html -s /path/to/source/folder",
	Short:   "Generate HTML formatted report",
	Example: execExampleToHTML,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logger.NewLogger(AppConfig, "core")
		logger.Info("to-html called")

		sarifReport, err := scaniosarif.ReadReport(allToHTMLOptions.Input, logger, allToHTMLOptions.SourceFolder)
		if err != nil {
			return err
		}

		sarifReport.EnrichResultsProperties()

		repositoryMetadata, err := collectRepositoryMetadata()
		if err != nil {
			logger.Debug("can't collect repository metadata", "err", err)
		}

		toolMetadata, err := sarifReport.ExtractToolNameAndVersion()
		if err != nil {
			return err
		}

		severityInfo := sarifReport.CollectSeverityInfo()

		metadata := &ReportMetadata{
			RepositoryMetadata: *repositoryMetadata,
			ToolMetadata:       *toolMetadata,
			Title:              allToHTMLOptions.Title,
			Time:               time.Now().UTC(),
			SourceFolder:       allToHTMLOptions.SourceFolder,
			SeverityInfo:       severityInfo,
		}

		sarifReport.SortResultsByLevel()
		sarifReport.RemoveDataflowDuplicates()

		tmpl, err := template.New("report.html").
			Funcs(template.FuncMap{
				"add":              add,
				"generateSequence": generateSequence,
				"formatDateTime":   formatDateTime,
			}).
			ParseFiles(filepath.Join(allToHTMLOptions.TempatesPath, "report.html"))
		if err != nil {
			return err
		}

		data := struct {
			Metadata *ReportMetadata
			Report   *sarif.Report
		}{
			Metadata: metadata,
			Report:   sarifReport.Report,
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

	toHtmlCmd.Flags().StringVar(&allToHTMLOptions.TempatesPath, "templates-path", "./templates/tohtml", "path to folder with templates")
	toHtmlCmd.Flags().StringVar(&allToHTMLOptions.Title, "title", "Scanio Report", "title for generated html file")
	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.Input, "input", "i", "", "input file with sarif report")
	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.OutputFile, "output", "o", "scanio-report.html", "outoput file")
	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.SourceFolder, "source", "s", "", "source folder")
}
