package cmd

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/spf13/cobra"
)

type ToHTMLOptions struct {
	TempatesPath string
	Title        string
	OutputFile   string
	Input        string
}

var allToHTMLOptions ToHTMLOptions

func enrichResultsProperties(sarifReport *sarif.Report) {
	rulesMap := map[string]*sarif.ReportingDescriptor{}
	for _, rule := range sarifReport.Runs[0].Tool.Driver.Rules {
		rulesMap[rule.ID] = rule
	}

	for _, result := range sarifReport.Runs[0].Results {
		if rule, ok := rulesMap[*result.RuleID]; ok {
			result.Properties["Title"] = rule.ShortDescription.Text
		}
	}
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

		tmpl, err := template.ParseFiles(filepath.Join(allToHTMLOptions.TempatesPath, "report.html"))
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
}
