package semgrepShared

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type SemgrepReportText struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}

type SemgrepReportSarif struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}

type Run struct {
	Invocations []Invocation `json:"invocations"`
	Results     []Result     `json:"results"`
	Tool        struct {
		Driver struct {
			Name            string `json:"name"`
			SemanticVersion string `json:"semanticVersion"`
			Rules           []Rule `json:"rules"`
		} `json:"driver"`
	} `json:"tool"`
}

type Invocation struct {
	ExecutionSuccessful        bool           `json:"executionSuccessful"`
	ToolExecutionNotifications []Notification `json:"toolExecutionNotifications"`
}

type Notification struct {
	Descriptor struct {
		ID string `json:"id"`
	} `json:"descriptor"`
	Level   string `json:"level"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
}

type Rule struct {
	DefaultConfiguration struct {
		Level string `json:"level"`
	} `json:"defaultConfiguration"`
	FullDescription struct {
		Text string `json:"text"`
	} `json:"fullDescription"`
	Help             string         `json:"help"`
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Properties       RuleProperties `json:"properties"`
	ShortDescription struct {
		Text string `json:"text"`
	} `json:"shortDescription"`
}

type RuleProperties struct {
	Precision        string   `json:"precision"`
	Tags             []string `json:"tags"`
	SecuritySeverity string   `json:"security-severity"`
}

type Result struct {
	RuleID  string `json:"ruleId"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
	Locations []struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI       string `json:"uri"`
				URIBaseID string `json:"uriBaseId"`
			} `json:"artifactLocation"`
			Region struct {
				StartLine   int `json:"startLine"`
				StartColumn int `json:"startColumn"`
				EndLine     int `json:"endLine"`
				EndColumn   int `json:"endColumn"`
			} `json:"region"`
		} `json:"physicalLocation"`
	} `json:"locations"`
	Suppressions []struct {
		Kind   string `json:"kind"`
		Status string `json:"status,omitempty"`
		GUID   string `json:"guid,omitempty"`
	} `json:"suppressions,omitempty"`
}

type Finding struct {
	FilePath    string
	VulnDetails []VulnDetail
}

type VulnDetail struct {
	RuleName     string
	Description  string
	DetailsURL   string
	Autofix      string
	CodeSnippets []string
}

type reportParser struct {
	scanner              *bufio.Scanner
	currentFinding       Finding
	descriptionLines     []string
	codeSnippetLines     []string
	inDescription        bool
	inCodeSnippet        bool
	findingsCount        int
	previousLine         string
	skipBlankSnippetLine bool
}

func newReportParser(scanner *bufio.Scanner) *reportParser {
	return &reportParser{
		scanner: scanner,
	}
}

func ParseSemgrepTextShort(filePath string, maxLength int) (string, bool, error) {
	var reportBuilder strings.Builder
	file, err := os.Open(filePath)
	if err != nil {
		return "", false, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return "", false, fmt.Errorf("error getting file stats: %v", err)
	}

	reportBuilder.WriteString("**Semgrep scanner resutls**\n")
	if fileInfo.Size() == 0 {
		reportBuilder.WriteString(fmt.Sprintf("The scanner found 0 issues.\n\n"))
		return reportBuilder.String(), true, nil
	}

	scanner := bufio.NewScanner(file)
	parser := newReportParser(scanner)

	count, counterLineNumber, err := parser.parseOnlyCount()
	if err != nil {
		return "", false, fmt.Errorf("error parsing report: %v", err)
	}

	// Use strings.Builder for efficient string concatenation

	reportBuilder.WriteString(fmt.Sprintf("The scanner found %d issues.\n\n", count))
	reportBuilder.WriteString("```\n")

	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)
	skipLines := counterLineNumber + 2
	for i := 0; i < skipLines; i++ {
		if !scanner.Scan() {
			break
		}
	}

	for scanner.Scan() {
		reportBuilder.WriteString(scanner.Text() + "\n")
		if reportBuilder.Len() > maxLength {
			reportBuilder.WriteString("\n```\n ⚠️ output was truncated\n")
			break
		}
	}

	//TODO
	//reportBuilder.WriteString("```\n")

	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("error reading the file: %v", err)
	}

	return reportBuilder.String(), false, nil
}

func (p *reportParser) parseOnlyCount() (int, int, error) {
	var (
		findingsNumber, lineNumber int
	)

	for p.scanner.Scan() {
		line := p.scanner.Text()
		line = strings.TrimSpace(line)
		lineNumber++

		if isFindingsCountLine(line) {
			count, err := extractFindingsCount(line)
			if err != nil {
				return findingsNumber, lineNumber, err
			}
			findingsNumber = count
			break
		}
	}

	return findingsNumber, lineNumber, nil
}

func isFindingsCountLine(line string) bool {
	return strings.Contains(line, "Code Findings")
}

func extractFindingsCount(line string) (int, error) {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 0 {
		return strconv.Atoi(matches[0])
	}
	return 0, fmt.Errorf("no findings count found in line: %s", line)
}

// an attempt to write a parser of a text report using a straightforward approach without a lexer
// func (p *reportParser) parse() ([]Finding, error) {
// 	var findings []Finding

// 	for p.scanner.Scan() {
// 		line := p.scanner.Text()
// 		if err := p.processLine(line); err != nil {
// 			return nil, err
// 		}

// 		if p.shouldStartNewFinding(line) && p.currentFinding.FilePath != "" {
// 			p.finalizeCurrentFinding()
// 			findings = append(findings, p.currentFinding)
// 			p.resetCurrentFinding()
// 		}
// 		p.previousLine = line
// 	}

// 	if err := p.scanner.Err(); err != nil {
// 		return nil, err
// 	}

// 	if p.currentFinding.FilePath != "" {
// 		p.finalizeCurrentFinding()
// 		findings = append(findings, p.currentFinding)
// 	}

// 	return findings, nil
// }

// func (p *reportParser) processLine(line string) error {
// 	line = strings.TrimSpace(line)

// 	if isFindingsCountLine(line) {
// 		count, err := extractFindingsCount(line)
// 		if err != nil {
// 			return err
// 		}
// 		p.findingsCount = count
// 		return nil
// 	}

// 	if p.currentFinding.FilePath == "" && isFilePath(line) {
// 		p.startNewFinding(line)
// 		return nil
// 	}

// 	currentVulnDetail := getLastVulnDetail(&p.currentFinding)

// 	if p.inDescription {
// 		if currentVulnDetail.RuleName == "" {
// 			currentVulnDetail.RuleName = line
// 			return nil
// 		}

// 		if strings.HasPrefix(line, "Details:") {
// 			currentVulnDetail.DetailsURL = strings.TrimPrefix(line, "Details:")
// 			currentVulnDetail.Description = strings.Join(p.descriptionLines, "\n")
// 			p.descriptionLines = nil
// 			p.inDescription = false
// 			p.inCodeSnippet = true
// 			p.skipBlankSnippetLine = true
// 			return nil
// 		}

// 		p.descriptionLines = append(p.descriptionLines, line)
// 		return nil
// 	}

// 	if p.inCodeSnippet {
// 		if line == "" && p.skipBlankSnippetLine {
// 			p.skipBlankSnippetLine = false
// 			return nil
// 		}
// 		if isCodeSnippetEnd(line) || (line == "" && !p.skipBlankSnippetLine) {
// 			p.codeSnippetLines = append(p.codeSnippetLines, line)

// 			if line == "" {
// 				p.finalizeCurrentFinding()
// 			}

// 			return nil
// 		}

// 		if p.skipBlankSnippetLine {
// 			if !isPartOfSnippet(line) {
// 				currentVulnDetail.CodeSnippets = append(currentVulnDetail.CodeSnippets, p.codeSnippetLines...)
// 				p.codeSnippetLines = nil
// 				p.inCodeSnippet = false
// 			}
// 			p.skipBlankSnippetLine = false
// 		}

// 		if p.inCodeSnippet {
// 			p.codeSnippetLines = append(p.codeSnippetLines, line)
// 		}

// 		return nil
// 	}

// 	return nil
// }

// func isPartOfSnippet(line string) bool {
// 	return strings.Contains(line, "┆")
// }

// func getLastVulnDetail(finding *Finding) *VulnDetail {
// 	if len(finding.VulnDetails) == 0 {
// 		finding.VulnDetails = append(finding.VulnDetails, VulnDetail{})
// 	}
// 	return &finding.VulnDetails[len(finding.VulnDetails)-1]
// }

// func (p *reportParser) shouldStartNewFinding(line string) bool {
// 	line = strings.TrimSpace(line)
// 	return ((line == "" && !p.inDescription && !p.inCodeSnippet && !p.skipBlankSnippetLine) || (p.previousLine == "⋮┆----------------------------------------" && len(line) > 0))
// }

// func (p *reportParser) startNewFinding(filePath string) {
// 	p.currentFinding.FilePath = filePath
// 	p.inDescription = true
// }

// func (p *reportParser) finalizeCurrentFinding() {
// 	if p.inCodeSnippet {
// 		lastVulnDetail := getLastVulnDetail(&p.currentFinding)
// 		lastVulnDetail.CodeSnippets = p.codeSnippetLines
// 	}
// 	p.codeSnippetLines = nil
// }

// func (p *reportParser) resetCurrentFinding() {
// 	p.currentFinding = Finding{}
// 	p.inDescription = true
// }

// func isFilePath(line string) bool {
// 	return strings.Contains(line, "/")
// }

// func isCodeSnippetEnd(line string) bool {
// 	return strings.Contains(line, "⋮┆----------------------------------------")
// }
