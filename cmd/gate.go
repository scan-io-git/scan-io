/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/google/cel-go/cel"
	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/spf13/cobra"
)

func doGate() {

	reportPath := "codeql-2023-06-17T15:01:17Z.sarifv2.1.0"
	jsonFile, err := os.Open(reportPath)
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	defer jsonFile.Close()

	var report sarif.Report
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal([]byte(byteValue), &report)

	env, _ := cel.NewEnv(
		cel.Variable("name", cel.StringType),
		cel.Variable("group", cel.StringType),
		cel.Variable("report", cel.MapType(cel.StringType, cel.AnyType)),
	)

	// ast, issues := env.Compile(`name.startsWith("/groups/" + group)`)
	ast, issues := env.Compile(`has(report.runs)`)
	if issues != nil && issues.Err() != nil {
		log.Fatalf("type-check error: %s", issues.Err())
	}
	prg, err := env.Program(ast)
	if err != nil {
		log.Fatalf("program construction error: %s", err)
	}

	out, details, err := prg.Eval(map[string]interface{}{
		"name":   "/groups/acme.co/documents/secret-stuff",
		"group":  "acme.co",
		"report": report,
	})
	fmt.Println(out)     // 'true'
	fmt.Println(details) // 'true'
	fmt.Println(err)     // 'true'
}

// gateCmd represents the gate command
var gateCmd = &cobra.Command{
	Use:   "gate",
	Short: "...",
	Long:  `...`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("gate called")
		doGate()
	},
}

func init() {
	rootCmd.AddCommand(gateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// gateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// gateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
