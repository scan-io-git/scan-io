package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/pkg/shared"
	cfg "github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
	"github.com/scan-io-git/scan-io/plugins/semgrep/pkg/shared"
	"github.com/scan-io-git/scan-io/plugins/trufflehog3/pkg/shared"
)

type DecisionMakerOptions struct {
	Scenario      string
	VCSPlugName   string
	VCSURL        string
	Namespace     string
	Repository    string
	PullRequestId int
	Login         string
}

type AppDecConfig struct {
	TemplatePath         string
	SemgrepRulesPath     string
	TruffleHog3RulesPath string
	ExecutionEnv         string
}

type HandlerError struct {
	Text  string
	Stage string
	Err   error
}

var (
	allDecisionMakerOptions       DecisionMakerOptions
	execExampleDecisionMaker      = `# scanio`
	message, status, templatePath string
	reportData                    = shared.ScanReportData{
		ScanStarted:  false,
		ScanPassed:   false,
		ScanFailed:   false,
		ScanCrashed:  false,
		ScanDetails:  "",
		ScanResults:  "",
		ErrorDetails: "",
	}
)

func ReadResultBuffer() []byte {
	shared.ResultBufferMutex.Lock()
	defer shared.ResultBufferMutex.Unlock()

	return shared.ResultBuffer.Bytes()
}

func LoadConfig() (*AppDecConfig, error) {
	config := &AppDecConfig{
		TemplatePath:         os.Getenv("SCANIO_TEMPLATE_PATH"),
		SemgrepRulesPath:     os.Getenv("SCANIO_SEMGREP_RULES"),
		TruffleHog3RulesPath: os.Getenv("SCANIO_TRUFFLEHOG3_RULES"),
	}

	if cfg.IsCI(AppConfig) {
		config.ExecutionEnv = os.Getenv("SCANIO_CI")

		switch config.ExecutionEnv {
		case "jenkins":
			buildUrl := os.Getenv("BUILD_URL")
			appendToScanDetails(fmt.Sprintf("Jenkins build URL of the job is %v.\n", buildUrl))
		default:
		}
	}

	rawStartTime := time.Now().UTC()
	startTime := rawStartTime.Format(time.RFC3339)
	appendToScanDetails(fmt.Sprintf("The job was started at %v.", startTime))
	if config.TemplatePath == "" || config.SemgrepRulesPath == "" || config.TruffleHog3RulesPath == "" {
		return nil, fmt.Errorf("missing required configuration")
	}

	return config, nil
}

func appendToScanDetails(newDetail string) {
	if existingDetails, ok := reportData.ScanDetails.(string); ok {
		reportData.ScanDetails = existingDetails + newDetail
	} else {
		reportData.ScanDetails = newDetail
	}
}

func executeCommand(command string, args []string) ([]shared.GenericResult, error) {
	buildCommand := append([]string{command}, args...)
	rootCmd.SetArgs(buildCommand)

	if err := rootCmd.Execute(); err != nil {
		return nil, fmt.Errorf("Failed to execute command. %v", err)
	}

	resultBufferContent := ReadResultBuffer()
	var result shared.GenericLaunchesResult

	if err := json.Unmarshal(resultBufferContent, &result); err != nil {
		return nil, fmt.Errorf("Failed to parse output. %v", err)
	}

	return result.Launches, nil
}

func (e HandlerError) Error() string {
	return fmt.Sprintf("error in %s stage | text: %s | error: %v", e.Stage, e.Text, e.Err)
}

func checkPR() ([]shared.GenericResult, error) {
	args := []string{
		"--vcs", allDecisionMakerOptions.VCSPlugName,
		"--action", "checkPR",
		"--vcs-url", allDecisionMakerOptions.VCSURL,
		"--namespace", allDecisionMakerOptions.Namespace,
		"--repository", allDecisionMakerOptions.Repository,
		"--pull-request-id", fmt.Sprintf("%v", allDecisionMakerOptions.PullRequestId),
	}

	result, err := executeCommand("integration-vcs", args)
	if err != nil {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "checkPR", Err: err}
	}

	if result[0].Status != "OK" {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "checkPR", Err: fmt.Errorf(result[0].Message)}
	}
	return result, nil
}

func addReviewer() ([]shared.GenericResult, error) {
	args := []string{
		"--vcs", allDecisionMakerOptions.VCSPlugName,
		"--action", "addRoleToPR",
		"--vcs-url", allDecisionMakerOptions.VCSURL,
		"--namespace", allDecisionMakerOptions.Namespace,
		"--repository", allDecisionMakerOptions.Repository,
		"--pull-request-id", fmt.Sprintf("%v", allDecisionMakerOptions.PullRequestId),
		"--login", allDecisionMakerOptions.Login,
		"--role", "REVIEWER",
	}

	result, err := executeCommand("integration-vcs", args)
	if err != nil {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "addReviewer", Err: err}
	}
	if result[0].Status != "OK" {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "addReviewer", Err: fmt.Errorf(result[0].Message)}
	}

	return result, nil
}

func changePRStatus(status string) ([]shared.GenericResult, error) {
	args := []string{
		"--vcs", allDecisionMakerOptions.VCSPlugName,
		"--action", "setStatusOfPR",
		"--vcs-url", allDecisionMakerOptions.VCSURL,
		"--namespace", allDecisionMakerOptions.Namespace,
		"--repository", allDecisionMakerOptions.Repository,
		"--pull-request-id", fmt.Sprintf("%v", allDecisionMakerOptions.PullRequestId),
		"--login", allDecisionMakerOptions.Login,
		"--status", status,
	}

	result, err := executeCommand("integration-vcs", args)
	if err != nil {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "changePRStatus", Err: err}
	}
	if result[0].Status != "OK" {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "changePRStatus", Err: fmt.Errorf(result[0].Message)}
	}

	return result, nil
}

func addComment(message string) ([]shared.GenericResult, error) {
	args := []string{
		"--vcs", allDecisionMakerOptions.VCSPlugName,
		"--action", "addComment",
		"--vcs-url", allDecisionMakerOptions.VCSURL,
		"--namespace", allDecisionMakerOptions.Namespace,
		"--repository", allDecisionMakerOptions.Repository,
		"--pull-request-id", fmt.Sprintf("%v", allDecisionMakerOptions.PullRequestId),
		"--comment", message,
	}

	result, err := executeCommand("integration-vcs", args)
	reportData.ScanStarted = false
	if err != nil {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "addComment", Err: err}
	}
	if result[0].Status != "OK" {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "addComment", Err: fmt.Errorf(result[0].Message)}
	}
	return result, nil
}

func fetchPR(fetchingLink string) ([]shared.GenericResult, error) {
	args := []string{
		"--vcs", "bitbucket",
		"--auth-type", "ssh-agent",
		fetchingLink,
	}
	result, err := executeCommand("fetch", args)

	if err != nil {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "fetchPR", Err: err}
	}
	if result[0].Status != "OK" {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "fetchPR", Err: fmt.Errorf(result[0].Message)}
	}
	return result, nil
}

func execScanner(scannerName, format, rulesPathSemgrep, resultsFolderPath, scaningFolder string, extraArgs []string) ([]shared.GenericResult, error) {
	baseArgs := []string{
		"--scanner", scannerName,
		"-c", rulesPathSemgrep,
		"--format", format,
		"--output", resultsFolderPath,
		scaningFolder,
	}
	allArgs := append(baseArgs, extraArgs...)
	result, err := executeCommand("analyse", allArgs)

	if err != nil {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "execScanner", Err: err}
	}
	if result[0].Status != "OK" {
		return nil, HandlerError{Text: "subcommand execution failed", Stage: "execScanner", Err: fmt.Errorf(result[0].Message)}
	}
	return result, nil
}

func predefinedPRHandler(logger hclog.Logger, config *AppDecConfig) error {
	//Ad-hoc predefined scenario for MVP
	//Base command: scanio decision-maker --scenario scanPR --vcs bitbucket --vcs-url git.com --namespace TEST --repository test --pull-request-id ID

	// checkPR: scanio integration-vcs --vcs bitbucket --action checkPR --vcs-url git.com --namespace TEST --repository test --pull-request-id ID
	checkPRResult, err := checkPR()
	if err != nil {
		return err
	}

	resultAsserted, ok := checkPRResult[0].Result.(map[string]interface{})
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for checkPRResult[0].Result",
			Stage: "checkPR",
			Err:   fmt.Errorf("expected type map[string]interface{}, got %T", checkPRResult[0].Result),
		}
	}

	fetchingLink := resultAsserted["SelfLink"].(string)
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for resultAsserted['SelfLink']",
			Stage: "checkPR",
			Err:   fmt.Errorf("expected type (string), got %T", resultAsserted["SelfLink"]),
		}
	}

	// add reviewer to the repo: scanio integration-vcs --vcs bitbucket --action addRoleToPR --vcs-url git.com --namespace TEST --repository test --pull-request-id ID --login --role REVIWER
	_, err = addReviewer()
	if err != nil {
		return err
	}

	// block the PR: scanio integration-vcs --vcs bitbucket --action setStatusOfPR --vcs-url git.com --namespace TEST --repository test --pull-request-id ID --login LOGIN --status UNAPPROVED
	_, err = changePRStatus("UNAPPROVED")
	if err != nil {
		return err
	}

	// add a comment to the PR: scanio integration-vcs --vcs bitbucket --action addComment --vcs-url git.com --namespace TEST --repository test --comment "Test text"
	reportData.ScanStarted = true
	startMessage, err := shared.CommentBuilder(reportData, config.TemplatePath)
	_, err = addComment(startMessage)
	if err != nil {
		return err
	}

	// fetch repo: scanio integration-vcs --vcs bitbucket --action checkPR --vcs-url git.com --namespace TEST --repository test --pull-request-id 9
	fetchingResult, err := fetchPR(fetchingLink)
	if err != nil {
		return err
	}

	argsAsserted, ok := fetchingResult[0].Args.(map[string]interface{})
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for fetchingResult[0].Args",
			Stage: "fetchPR",
			Err:   fmt.Errorf("expected type map[string]interface{}, got %T", fetchingResult[0].Args),
		}
	}

	repoParamAsserted, ok := argsAsserted["RepoParam"].(map[string]interface{})
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for argsAsserted['RepoParam']",
			Stage: "fetchPR",
			Err:   fmt.Errorf("expected type map[string]interface{}, got %T", argsAsserted["RepoParam"]),
		}
	}

	resultAsserted, ok = fetchingResult[0].Result.(map[string]interface{})
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for fetchingResult[0].Result",
			Stage: "fetchPR",
			Err:   fmt.Errorf("expected type map[string]interface{}, got %T", fetchingResult[0].Result),
		}
	}

	scaningFolder, ok := resultAsserted["Path"].(string)
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for fetchingResult[0].Result",
			Stage: "fetchPR",
			Err:   fmt.Errorf("expected type (string), got %T", resultAsserted["Path"]),
		}
	}

	vcsUrl := repoParamAsserted["vcs_url"].(string)
	namespace := repoParamAsserted["namespace"].(string)
	repoName := repoParamAsserted["repo_name"].(string)
	resultsFolderPath := filepath.Join(cfg.GetScanioResultsHome(AppConfig), strings.ToLower(vcsUrl), filepath.Join(strings.ToLower(namespace), strings.ToLower(repoName)))

	// scan code: scanio analyse --scanner semgrep --format sarif /[scanio]/test
	scanningResultSemgrep, err := execScanner("semgrep", "text", config.SemgrepRulesPath, resultsFolderPath, scaningFolder, []string{})
	if err != nil {
		return err
	}

	resultAsserted, ok = scanningResultSemgrep[0].Result.(map[string]interface{})
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for scanningResultSemgrep[0].Result",
			Stage: "execScanner",
			Err:   fmt.Errorf("expected type map[string]interface{}, got %T", scanningResultSemgrep[0].Result),
		}
	}

	semgrepReportPath, ok := resultAsserted["ResultsPath"].(string)
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for resultAsserted['ResultsPath']",
			Stage: "execScanner",
			Err:   fmt.Errorf("expected type (string), got %T", resultAsserted["ResultsPath"]),
		}
	}

	// scan code: scanio analyse --scanner trufflehog3 -c /scanio-rules/trufflehog-rules/rules.yaml /[scanio]/test -- --no-history
	extraArgs := []string{"--", "--no-history"}
	scanningResultTrufflehog3, err := execScanner("trufflehog3", "json", config.TruffleHog3RulesPath, resultsFolderPath, scaningFolder, extraArgs)
	if err != nil {
		return err
	}

	resultAsserted, ok = scanningResultTrufflehog3[0].Result.(map[string]interface{})
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for scanningResultTrufflehog3[0].Result",
			Stage: "execScanner",
			Err:   fmt.Errorf("expected type map[string]interface{}, got %T", scanningResultTrufflehog3[0].Result),
		}
	}

	trufflehog3ReportPath, ok := resultAsserted["ResultsPath"].(string)
	if !ok {
		return HandlerError{Text: "assertion error for results", Stage: "execScanner", Err: fmt.Errorf("")}
	}
	if !ok {
		return HandlerError{
			Text:  "type assertion failed for resultAsserted['ResultsPath']",
			Stage: "execScanner",
			Err:   fmt.Errorf("expected type (string), got %T", resultAsserted["ResultsPath"]),
		}
	}

	// ad-hoc parsing reports
	var finalStatus string
	maxLength := 3000
	reportTextSemgrep, verdictSemgrep, err := semgrepShared.ParseSemgrepTextShort(semgrepReportPath, maxLength)
	if err != nil {
		return HandlerError{Text: "semgrep report parsing failed", Stage: "buildReport", Err: err}
	}

	reportTextTrufflehog3, verdictTrufflehog3, err := trufflehog3Shared.ParseTrufflehog3Json(trufflehog3ReportPath, maxLength)
	if err != nil {
		return HandlerError{Text: "trufflehog3 report parsing failed", Stage: "buildReport", Err: err}
	}

	combinedReports := reportTextSemgrep + "\n============================\n\n" + reportTextTrufflehog3
	reportData.ScanResults = combinedReports

	if !verdictSemgrep || !verdictTrufflehog3 {
		finalStatus = "NEEDWORK"
		reportData.ScanFailed = true
	} else {
		finalStatus = "APPROVED"
		reportData.ScanPassed = true
	}

	message, err := shared.CommentBuilder(reportData, config.TemplatePath)
	if err != nil {
		return HandlerError{Text: "error building a comment", Stage: "buildReport", Err: err}
	}

	// add a comment to the PR: scanio integration-vcs --vcs bitbucket --action addComment --vcs-url git.com --namespace TEST --repository test --comment ""
	_, err = addComment(message)
	if err != nil {
		return err
	}

	if reportData.ScanPassed == true {
		// unblock the PR: scanio integration-vcs --vcs bitbucket --action setStatusOfPR --vcs-url git.com --namespace TEST --repository test --pull-request-id ID --login LOGIN --status UNAPPROVED
		_, err = changePRStatus(finalStatus)
		if err != nil {
			return err
		}
	}

	return nil
}

func scanioHandler(logger hclog.Logger, scenario string) error {
	config, err := LoadConfig()
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		return err
	}

	switch scenario {
	case "scanPR":
		if err := predefinedPRHandler(logger, config); err != nil {
			reportData.ScanCrashed = true
			reportData.ErrorDetails = err.Error()

			message, _ := shared.CommentBuilder(reportData, config.TemplatePath)
			// if err != nil {
			// 	return HandlerError{Text: "subcommand execution failed", Stage: "Main scenario handler addComment", Err: err}
			// }

			_, _ = addComment(message)
			// if err != nil {
			// 	return HandlerError{Text: "subcommand execution failed", Stage: "Main scenario handler addComment", Err: err}

			// }
			// commentingPRResultStatus := commentingPRResult[0].Status
			// if commentingPRResultStatus != "OK" {
			// 	errorMessage := commentingPRResult[0].Message
			// 	return fmt.Errorf("The subcommand special handler execution is failed %v", errorMessage)
			// }

			return fmt.Errorf("predefined scenario crashed: %v", err)
		}
	default:
		return fmt.Errorf("scenario not implemented: %s", scenario)
	}

	return nil
}

var handlerCmd = &cobra.Command{
	Use:                   "decision-maker --scenario SCENARIO_NAME ...",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               execExampleDecisionMaker,
	Short:                 "[EXPERIMENTAL] For the specific ad-hoc scenario, not for production",
	Long:                  `[EXPERIMENTAL] For the specific ad-hoc scenario, not for production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logger.NewLogger(AppConfig, "core-decision-maker")

		checkArgs := func() error {
			if len(allDecisionMakerOptions.Scenario) == 0 {
				return fmt.Errorf("The 'action' flag must be specified!")
			}
			return nil
		}

		if err := checkArgs(); err != nil {
			return err
		}

		switch allDecisionMakerOptions.Scenario {
		case "scanPR":
			logger.Info("Executing scanPR scenario")
			if err := scanioHandler(logger, allDecisionMakerOptions.Scenario); err != nil {
				return fmt.Errorf("Failed to execute scanPR scenario: %v", err)
			}
			return nil
		default:
			return fmt.Errorf("The scenarion is not implemented %v", allDecisionMakerOptions.Scenario)
		}
	},
}

func init() {
	rootCmd.AddCommand(handlerCmd)

	handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Scenario, "scenario", "", "Predifined scenario for handling a bunch of Scanio commands. Eg. scanPR, etc.")

	handlerCmd.Flags().StringVar(&allDecisionMakerOptions.VCSPlugName, "vcs", "", "the plugin name of the VCS used. Eg. bitbucket, gitlab, github, etc.")
	handlerCmd.Flags().StringVar(&allDecisionMakerOptions.VCSURL, "vcs-url", "", "URL to a root of the VCS API. Eg. github.com.")
	handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Namespace, "namespace", "", "the name of a specific namespace. Namespace for Gitlab is an organization, for Bitbucket_v1 is a project.")
	handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Repository, "repository", "", "the name of a specific repository.")
	handlerCmd.Flags().IntVar(&allDecisionMakerOptions.PullRequestId, "pull-request-id", 0, "the id of specific PR form the repository.")
	handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Login, "login", "", "login for integrations. For example, add reviewer with this login to PR.")
}
