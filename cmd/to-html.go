package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
)

type ToHTMLOptions struct {
	TempatesPath string
	Title        string
	OutputFile   string
	Input        string
	SourceFolder string
}

var allToHTMLOptions ToHTMLOptions

func enrichResultsTitleProperty(sarifReport *sarif.Report) {
	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range sarifReport.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range sarifReport.Runs[0].Results {
		if rule, ok := rulesMap[*result.RuleID]; ok {
			if result.Properties == nil {
				result.Properties = make(map[string]interface{})
			}
			result.Properties["Title"] = rule.ShortDescription.Text
		}
	}
}

func readLineFromFile(loc *sarif.PhysicalLocation) (string, error) {
	//return error if allToHTMLOptions.SourceFolder is not specified
	if allToHTMLOptions.SourceFolder == "" {
		return "", fmt.Errorf("source folder is not set")
	}

	// Construct the file path
	filePath := filepath.Join(allToHTMLOptions.SourceFolder, *loc.ArtifactLocation.URI)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	currentLine := 0
	for scanner.Scan() {
		currentLine++
		if currentLine == *loc.Region.StartLine {
			return scanner.Text(), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return "", fmt.Errorf("line %d not found in file", *loc.Region.StartLine)
}

// function to enrich location properties with code and URI
func enrichResultsLocationProperty(location *sarif.Location) error {
	artifactLocation := location.PhysicalLocation.ArtifactLocation
	if artifactLocation.Properties == nil {
		artifactLocation.Properties = make(map[string]interface{})
	}
	artifactLocation.Properties["URI"] = *artifactLocation.URI

	// return if allToHTMLOptions.SourceFolder is not specified
	if allToHTMLOptions.SourceFolder == "" {
		return fmt.Errorf("source folder is not set")
	}

	codeLine, err := readLineFromFile(location.PhysicalLocation)
	if err != nil {
		return err
	}
	// print amount of spaces bnefore code
	// spacePrefixLength := len(codeLine) - len(strings.TrimLeft(codeLine, " "))
	// artifactLocation.Properties["Code"] = strings.TrimLeft(codeLine, " ")
	artifactLocation.Properties["Code"] = codeLine
	spacePrefixLength := 0

	if location.PhysicalLocation.Region.Properties == nil {
		location.PhysicalLocation.Region.Properties = make(map[string]interface{})
	}
	if location.PhysicalLocation.Region.StartColumn != nil {
		location.PhysicalLocation.Region.Properties["StartColumn"] = *location.PhysicalLocation.Region.StartColumn - spacePrefixLength - 1
	} else {
		location.PhysicalLocation.Region.Properties["StartColumn"] = 0
	}
	if location.PhysicalLocation.Region.EndColumn != nil {
		location.PhysicalLocation.Region.Properties["EndColumn"] = *location.PhysicalLocation.Region.EndColumn - spacePrefixLength - 1
	} else {
		location.PhysicalLocation.Region.Properties["EndColumn"] = 0
	}
	if location.PhysicalLocation.Region.StartLine != nil {
		location.PhysicalLocation.Region.Properties["StartLine"] = *location.PhysicalLocation.Region.StartLine - spacePrefixLength
	} else {
		location.PhysicalLocation.Region.Properties["StartLine"] = 0
	}
	if location.PhysicalLocation.Region.EndLine != nil {
		location.PhysicalLocation.Region.Properties["EndLine"] = *location.PhysicalLocation.Region.EndLine
	} else {
		location.PhysicalLocation.Region.Properties["EndLine"] = location.PhysicalLocation.Region.Properties["StartLine"]
	}

	return nil
}

func enrichResultsCodeFlowProperty(sarifReport *sarif.Report) {
	// init logger
	logger := shared.NewLogger("core")

	for _, result := range sarifReport.Runs[0].Results {

		if len(result.CodeFlows) == 0 && len(result.Locations) > 0 {
			//add new code flow
			codeFlow := sarif.NewCodeFlow()
			for _, location := range result.Locations {
				threadFlow := sarif.NewThreadFlow()
				threadFlow.Locations = append(threadFlow.Locations, &sarif.ThreadFlowLocation{
					Location: location,
				})
				codeFlow.ThreadFlows = append(codeFlow.ThreadFlows, threadFlow)
			}
			result.CodeFlows = append(result.CodeFlows, codeFlow)
		}

		for _, codeflow := range result.CodeFlows {
			for _, threadflow := range codeflow.ThreadFlows {
				for _, location := range threadflow.Locations {
					err := enrichResultsLocationProperty(location.Location)
					if err != nil {
						logger.Debug("can't read source file", "err", err)
						continue
					}
				}
			}
		}
	}
}

// function to enrich results properties with level taken from corersponding rules propertiues "problem.severity" field
func enrichResultsLevelProperty(sarifReport *sarif.Report) {
	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range sarifReport.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range sarifReport.Runs[0].Results {
		if rule, ok := rulesMap[*result.RuleID]; ok {
			if result.Properties["Level"] == nil {
				if result.Level != nil {
					// used by snyk
					result.Properties["Level"] = *result.Level
				} else if rule.Properties["problem.severity"] != nil {
					// used by codeql
					result.Properties["Level"] = rule.Properties["problem.severity"]
				} else if rule.DefaultConfiguration != nil {
					// used by all tools?
					result.Properties["Level"] = rule.DefaultConfiguration.Level
				} else {
					// just a fallback, should never happen
					result.Properties["Level"] = "unknown"
				}
			}
		}
	}
}

func enrichResultsProperties(sarifReport *sarif.Report) {
	enrichResultsTitleProperty(sarifReport)
	enrichResultsCodeFlowProperty(sarifReport)
	enrichResultsLevelProperty(sarifReport)
}

func add(a, b int) int {
	return a + b
}

// generateSequence generates a slice of integers from 1 to n.
func generateSequence(n int) []int {
	var sequence []int
	for i := 1; i <= n; i++ {
		sequence = append(sequence, i)
	}
	return sequence
}

// reads a sarif report from Input file
func readSarifReport() (*sarif.Report, error) {
	jsonFile, err := os.Open(allToHTMLOptions.Input)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	var sarifReport sarif.Report
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal([]byte(byteValue), &sarifReport)

	return &sarifReport, nil
}

// function that finds a path to git repository from the source folder
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

type ToolMetadata struct {
	Name    string
	Version *string
}

type ReportMetadata struct {
	RepositoryMetadata
	ToolMetadata
	Title        string
	Time         time.Time
	SourceFolder string
}

// function to collect metadata about the repository
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
func formatDateTime(t time.Time) string {
	day := ordinalDate(t.Day())
	return fmt.Sprintf("%s %s %d %d:%02d:%02d %s", day, t.Month(), t.Year(), t.Hour()%12, t.Minute(), t.Second(), t.Format("pm"))
}

// extract Tool name an version from sarifreport
func extractToolNameAndVersion(sarifReport *sarif.Report) (*ToolMetadata, error) {
	toolName := sarifReport.Runs[0].Tool.Driver.Name
	toolVersion := sarifReport.Runs[0].Tool.Driver.SemanticVersion
	return &ToolMetadata{
		Name:    toolName,
		Version: toolVersion,
	}, nil
}

// toHtmlCmd represents the toHtml command
var toHtmlCmd = &cobra.Command{
	Use:   "to-html",
	Short: "Generate HTML formatted report",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := shared.NewLogger("core")
		logger.Info("to-html called")

		sarifReport, err := readSarifReport()
		if err != nil {
			return err
		}

		enrichResultsProperties(sarifReport)

		repositoryMetadata, err := collectRepositoryMetadata()
		if err != nil {
			logger.Debug("can't collect repository metadata", "err", err)
		}

		ToolMetadata, err := extractToolNameAndVersion(sarifReport)
		if err != nil {
			return err
		}

		metadata := &ReportMetadata{
			RepositoryMetadata: *repositoryMetadata,
			ToolMetadata:       *ToolMetadata,
			Title:              allToHTMLOptions.Title,
			Time:               time.Now().UTC(),
			SourceFolder:       allToHTMLOptions.SourceFolder,
		}

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

	toHtmlCmd.Flags().StringVar(&allToHTMLOptions.TempatesPath, "templates-path", "./templates/tohtml", "path to folder with templates")
	toHtmlCmd.Flags().StringVar(&allToHTMLOptions.Title, "title", "Scanio Report", "title for generated html file")
	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.Input, "input", "i", "", "input file with sarif report")
	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.OutputFile, "output", "o", "scanio-report.html", "outoput file")
	toHtmlCmd.Flags().StringVarP(&allToHTMLOptions.SourceFolder, "source", "s", "", "source folder")
}
