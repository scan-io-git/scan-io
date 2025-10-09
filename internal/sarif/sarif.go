package sarif

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/hashicorp/go-hclog"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

type Report struct {
	*sarif.Report
	logger       hclog.Logger
	sourceFolder string
}

type ToolMetadata struct {
	Name    string
	Version *string
}

func readSarifReport(inputPath string) (*sarif.Report, error) {
	jsonFile, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	var sarifReport sarif.Report
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal([]byte(byteValue), &sarifReport)

	return &sarifReport, nil
}

// remove all results with Suppressions property
func removeSuppressedResults(report *sarif.Report) {
	for _, run := range report.Runs {
		var filteredResults []*sarif.Result

		for _, result := range run.Results {
			if len(result.Suppressions) == 0 {
				filteredResults = append(filteredResults, result)
			}
		}

		run.Results = filteredResults
	}
}

func ReadReport(inputPath string, logger hclog.Logger, sourceFolder string, noSuppressions bool) (*Report, error) {

	sarifReport, err := readSarifReport(inputPath)
	if err != nil {
		return nil, err
	}

	if noSuppressions {
		removeSuppressedResults(sarifReport)
	}

	// make an absolute path of source folder
	expandedSourceFolder, err := files.ExpandPath(sourceFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to expand source folder: %w", err)
	}
	absPath, err := filepath.Abs(expandedSourceFolder)
	if err != nil {
		return nil, err
	}

	return &Report{
		Report:       sarifReport,
		logger:       logger,
		sourceFolder: absPath,
	}, nil
}

// ExtractToolNameAndVersion function extracts tool name and version from a sarif report
func (r Report) ExtractToolNameAndVersion() (*ToolMetadata, error) {
	toolName := r.Runs[0].Tool.Driver.Name
	toolVersion := r.Runs[0].Tool.Driver.SemanticVersion
	return &ToolMetadata{
		Name:    toolName,
		Version: toolVersion,
	}, nil
}

// function that collects information about amount of low, mediumn and high severity issues
// returns a map with this information, and a total amount of issues
func (r Report) CollectSeverityInfo() map[string]int {
	severityInfo := map[string]int{
		"low":    0,
		"medium": 0,
		"high":   0,
		"total":  0,
	}

	for _, run := range r.Runs {
		for _, result := range run.Results {
			severity := result.Properties["Level"].(string)
			switch severity {
			case "error":
				severityInfo["high"]++
			case "warning":
				severityInfo["medium"]++
			default:
				severityInfo["low"]++
			}
			severityInfo["total"]++
		}
	}

	return severityInfo
}

// EnrichResultsTitleProperty function enriches sarif results properties with title and description values
func (r Report) EnrichResultsTitleProperty() {
	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range r.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range r.Runs[0].Results {
		if rule, ok := rulesMap[*result.RuleID]; ok {
			if result.Properties == nil {
				result.Properties = make(map[string]interface{})
			}
			if rule.ShortDescription != nil {
				result.Properties["Title"] = rule.ShortDescription.Text
			}
			if rule.FullDescription != nil && rule.FullDescription.Text != nil {
				result.Properties["Description"] = *rule.FullDescription.Text
			} else if result.Message.Text != nil {
				result.Properties["Description"] = *result.Message.Text
			}
		}
	}
}

// EnrichResultsLocationProperty function enriches sarif location properties with source code and URI values
func (r Report) EnrichResultsLocationProperty(location *sarif.Location) error {
	artifactLocation := location.PhysicalLocation.ArtifactLocation
	if artifactLocation.Properties == nil {
		artifactLocation.Properties = make(map[string]interface{})
	}

	// set artifactLocation.Properties["URI"] to be *artifactLocation.URI if it's a relative path,
	// otherwise trim prefix of r.sourceFolder from *artifactLocation.URI
	if !filepath.IsAbs(*artifactLocation.URI) {
		artifactLocation.Properties["URI"] = *artifactLocation.URI
	} else {
		artifactLocation.Properties["URI"] = (*artifactLocation.URI)[len(r.sourceFolder):]
		// remove slash if string start with slash
		if len(artifactLocation.Properties["URI"].(string)) > 0 && artifactLocation.Properties["URI"].(string)[0] == '/' {
			artifactLocation.Properties["URI"] = artifactLocation.Properties["URI"].(string)[1:]
		}
	}

	if location.PhysicalLocation.Region.Properties == nil {
		location.PhysicalLocation.Region.Properties = make(map[string]interface{})
	}
	if location.PhysicalLocation.Region.StartColumn != nil {
		location.PhysicalLocation.Region.Properties["StartColumn"] = *location.PhysicalLocation.Region.StartColumn - 1
	} else {
		location.PhysicalLocation.Region.Properties["StartColumn"] = 0
	}
	if location.PhysicalLocation.Region.EndColumn != nil {
		location.PhysicalLocation.Region.Properties["EndColumn"] = *location.PhysicalLocation.Region.EndColumn - 1
	} else {
		location.PhysicalLocation.Region.Properties["EndColumn"] = 0
	}
	if location.PhysicalLocation.Region.StartLine != nil {
		location.PhysicalLocation.Region.Properties["StartLine"] = *location.PhysicalLocation.Region.StartLine
	} else {
		location.PhysicalLocation.Region.Properties["StartLine"] = 0
	}
	if location.PhysicalLocation.Region.EndLine != nil {
		location.PhysicalLocation.Region.Properties["EndLine"] = *location.PhysicalLocation.Region.EndLine
	} else {
		location.PhysicalLocation.Region.Properties["EndLine"] = location.PhysicalLocation.Region.Properties["StartLine"]
	}

	// return if allToHTMLOptions.SourceFolder is not specified
	if r.sourceFolder == "" {
		return fmt.Errorf("source folder is not set")
	}
	codeLine, err := r.readLineFromFile(location.PhysicalLocation)
	if err != nil {
		return err
	}
	// print amount of spaces bnefore code
	// spacePrefixLength := len(codeLine) - len(strings.TrimLeft(codeLine, " "))
	// artifactLocation.Properties["Code"] = strings.TrimLeft(codeLine, " ")
	artifactLocation.Properties["Code"] = codeLine

	return nil
}

// readLineFromFile function reads a line from a file by the given location
func (r Report) readLineFromFile(loc *sarif.PhysicalLocation) (string, error) {
	//return error if allToHTMLOptions.SourceFolder is not specified
	if r.sourceFolder == "" {
		return "", fmt.Errorf("source folder is not set")
	}

	// Construct the file path
	// Use *loc.ArtifactLocation.URI value if it's an absolute path, and make a concatenation of r.sourceFolder and *loc.ArtifactLocation.URI otherwise
	filePath := *loc.ArtifactLocation.URI
	if !filepath.IsAbs(filePath) {
		fixedFilePath, err := files.ExpandPath(filepath.Join(r.sourceFolder, *loc.ArtifactLocation.URI))
		if err != nil {
			return "", fmt.Errorf("failed to contruct a file path: %w", err)
		}
		filePath = fixedFilePath
	}

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

// EnrichResultsCodeFlowProperty function enriches code flow location properties with source code and URI values
func (r Report) EnrichResultsCodeFlowProperty(locationWebURLCallback func(artifactLocation *sarif.Location) string) {

	for _, result := range r.Runs[0].Results {
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
					err := r.EnrichResultsLocationProperty(location.Location)
					if err != nil {
						r.logger.Debug("can't read source file", "err", err)
						continue
					}

					if location.Location.Properties == nil {
						location.Location.Properties = make(map[string]interface{})
					}
					location.Location.Properties["WebURL"] = locationWebURLCallback(location.Location)
				}
			}
		}
	}
}

// EnrichResultsLevelProperty function to enrich results properties with level taken from corersponding rules propertiues "problem.severity" field
func (r Report) EnrichResultsLevelProperty() {
	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range r.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range r.Runs[0].Results {
		if result.Properties == nil {
			result.Properties = make(map[string]interface{})
		}
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

func (r Report) EnrichResultsLocationURIProperty(locationWebURLCallback func(artifactLocation *sarif.Location) string) {
	for _, result := range r.Runs[0].Results {
		// if result location length is at least 1
		if len(result.Locations) > 0 {
			// get the first location
			location := result.Locations[0]
			// get the artifact location
			artifactLocation := location.PhysicalLocation.ArtifactLocation
			// if the artifact location has a URI
			if artifactLocation.URI != nil {
				// set the URI to the artifact location properties
				// set artifactLocation.Properties["URI"] to be *artifactLocation.URI if it's a relative path,
				// otherwise trim prefix of r.sourceFolder from *artifactLocation.URI
				if !filepath.IsAbs(*artifactLocation.URI) {
					artifactLocation.Properties["URI"] = *artifactLocation.URI
				} else {
					artifactLocation.Properties["URI"] = (*artifactLocation.URI)[len(r.sourceFolder):]
					// remove slash if string start with slash
					if len(artifactLocation.Properties["URI"].(string)) > 0 && artifactLocation.Properties["URI"].(string)[0] == '/' {
						artifactLocation.Properties["URI"] = artifactLocation.Properties["URI"].(string)[1:]
					}
				}

				if location.Properties == nil {
					location.Properties = make(map[string]interface{})
				}
				location.Properties["WebURL"] = locationWebURLCallback(location)
			}
		}
	}
}

// SortResultsByLevel function sorts sarif results by level
func (r Report) SortResultsByLevel() {

	for _, run := range r.Runs {
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
		sort.Slice(run.Results, func(i, j int) bool {
			return levelOrder[run.Results[i].Properties["Level"].(string)] < levelOrder[run.Results[j].Properties["Level"].(string)]
		})
	}
}

// remove codeflow duplicates
// each codeflow may have multiple threatflows. These threatflows may be equal for different codeflows.
// This function removes duplicates from codeflows
// if the codeflow is empty, it is removed
func (r Report) RemoveDataflowDuplicates() {
	for _, run := range r.Runs {
		for _, result := range run.Results {
			uniqueThreadFlowsFingerprints := map[string]bool{}
			for _, codeFlow := range result.CodeFlows {
				uniqueThreadFlows := []*sarif.ThreadFlow{}
				for _, threadFlow := range codeFlow.ThreadFlows {
					fingerprint := calculateThreadFlowFingerprint(threadFlow)
					if _, ok := uniqueThreadFlowsFingerprints[fingerprint]; !ok {
						uniqueThreadFlowsFingerprints[fingerprint] = true
						uniqueThreadFlows = append(uniqueThreadFlows, threadFlow)
					}
				}
				codeFlow.ThreadFlows = uniqueThreadFlows
			}

			// remove empty codeflows
			nonEmptyCodeFlows := []*sarif.CodeFlow{}
			for _, codeFlow := range result.CodeFlows {
				if len(codeFlow.ThreadFlows) > 0 {
					nonEmptyCodeFlows = append(nonEmptyCodeFlows, codeFlow)
				}
			}
			result.CodeFlows = nonEmptyCodeFlows
		}
	}
}

// function that calculates a fingerprint for threadflow
func calculateThreadFlowFingerprint(threadFlow *sarif.ThreadFlow) string {
	var fingerprint string
	for _, location := range threadFlow.Locations {
		fingerprint += fmt.Sprintf("|%s:%d:%d:%d:%d;",
			location.Location.PhysicalLocation.ArtifactLocation.Properties["URI"].(string),
			location.Location.PhysicalLocation.Region.Properties["StartLine"].(int),
			location.Location.PhysicalLocation.Region.Properties["StartColumn"].(int),
			location.Location.PhysicalLocation.Region.Properties["EndLine"].(int),
			location.Location.PhysicalLocation.Region.Properties["EndColumn"].(int),
		)
	}
	return calculateMD5Hash(fingerprint)
}

// function that calculates md5 hash for a given text
func calculateMD5Hash(text string) string {
	hash := md5.New()
	io.WriteString(hash, text)
	return hex.EncodeToString(hash.Sum(nil))
}
