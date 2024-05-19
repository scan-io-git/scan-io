package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

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

	codeLine, err := readLineFromFile(location.PhysicalLocation)
	if err != nil {
		return err
	}
	artifactLocation.Properties["Code"] = codeLine
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
						logger.Warn("can't read source file", "err", err)
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

// toHtmlCmd represents the toHtml command
var toHtmlCmd = &cobra.Command{
	Use:   "to-html",
	Short: "Generate HTML formatted report",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("toHtml called")

		jsonFile, err := os.Open(allToHTMLOptions.Input)
		if err != nil {
			return err
		}
		defer jsonFile.Close()

		var sarifReport sarif.Report
		byteValue, _ := io.ReadAll(jsonFile)
		json.Unmarshal([]byte(byteValue), &sarifReport)

		enrichResultsProperties(&sarifReport)

		tmpl, err := template.New("report.html").Funcs(template.FuncMap{"add": add, "generateSequence": generateSequence}).ParseFiles(filepath.Join(allToHTMLOptions.TempatesPath, "report.html"))
		if err != nil {
			return err
		}

		data := struct {
			Title  string
			Report sarif.Report
		}{
			Title:  allToHTMLOptions.Title,
			Report: sarifReport,
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
