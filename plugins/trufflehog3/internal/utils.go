package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Trufflehog3Rule struct {
	ID       string `json:"id"`
	Message  string `json:"message"`
	Pattern  string `json:"pattern"`
	Severity string `json:"severity"`
}

type Trufflehog3Issue struct {
	Rule   *Trufflehog3Rule `json:"rule"`
	Path   string           `json:"path"`
	Line   string           `json:"line"`
	Secret string           `json:"secret"`
}

type Trufflehog3Report []*Trufflehog3Issue

func (tf Trufflehog3Report) Render() string {
	triggers := map[string]Trufflehog3Report{}
	for _, i := range tf {
		all, ok := triggers[i.Path]
		if !ok {
			triggers[i.Path] = Trufflehog3Report{i}
		} else {
			triggers[i.Path] = append(all, i)
		}
	}
	report := []string{}
	for f, issues := range triggers {
		sort.Slice(issues, func(i, j int) bool {
			a := issues[i]
			b := issues[j]
			l1, _ := strconv.Atoi(a.Line)
			l2, _ := strconv.Atoi(b.Line)
			return l1 < l2
		})
		report = append(report, "```", f)
		report = append(report, fmt.Sprintf("Path: %v", f))
		for _, i := range issues {
			report = append(report, fmt.Sprintf(
				"    %s (%s severity) line %s: %s",
				i.Rule.ID,
				i.Rule.Severity,
				i.Line,
				i.Secret,
			))
		}
		report = append(report, "```")
	}
	return strings.Join(report, "\n")
}

func jsonToPlainReport(filePath string) (string, error) {
	var reportBuilder strings.Builder
	reportTrufflehog3 := Trufflehog3Report{}
	fileTrufflehog3, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %v", err)
	}
	reportBuilder.WriteString("**Trufflehog3 scanner resutls**\n")

	err = json.Unmarshal(fileTrufflehog3, &reportTrufflehog3)
	if err != nil {
		return "", fmt.Errorf("error parsing report: %v", err)
	}

	reportBuilder.WriteString(fmt.Sprintf("The scanner found %d issues.\n\n", len(reportTrufflehog3)))

	preReport := reportTrufflehog3.Render()
	reportBuilder.WriteString(preReport)
	finalReport := reportBuilder.String()
	reportBuilder.WriteString("```\n")

	return finalReport, nil
}
