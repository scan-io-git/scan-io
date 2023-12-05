package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
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

var (
	allDecisionMakerOptions  DecisionMakerOptions
	execExampleDecisionMaker = `# scanio`
)

func ReadResultBuffer() []byte {
	shared.ResultBufferMutex.Lock()
	defer shared.ResultBufferMutex.Unlock()

	return shared.ResultBuffer.Bytes()
}

func executeCommand(command string, args ...string) ([]shared.GenericResult, error) {
	buildCommand := append([]string{command}, args...)
	rootCmd.SetArgs(buildCommand)

	if err := rootCmd.Execute(); err != nil {
		return nil, fmt.Errorf("Failed to execute %s command: %w", buildCommand, err)
	}
	resultBufferContent := ReadResultBuffer()

	var resultOfSubcommand shared.GenericLaunchesResult
	if err := json.Unmarshal(resultBufferContent, &resultOfSubcommand); err != nil {
		return nil, fmt.Errorf("Failed to parse integration-vcs output for %s command: %w ", buildCommand, err)
	}

	return resultOfSubcommand.Launches, nil
}

func scanioHandler(logger hclog.Logger) error {
	//Base command: scanio decision-maker --scenario scanPR --vcs bitbucket --vcs-url git.com --namespace TEST --repository test --pull-request-id ID

	// #1 CheckPR: scanio integration-vcs --vcs bitbucket --action checkPR --vcs-url git.com --namespace TEST --repository test --pull-request-id ID
	checkPRResult, err := executeCommand("integration-vcs",
		"--vcs", allDecisionMakerOptions.VCSPlugName,
		"--action", "checkPR",
		"--vcs-url", allDecisionMakerOptions.VCSURL,
		"--namespace", allDecisionMakerOptions.Namespace,
		"--repository", allDecisionMakerOptions.Repository,
		"--pull-request-id", fmt.Sprintf("%v", allDecisionMakerOptions.PullRequestId),
	)
	if err != nil {
		return fmt.Errorf("The subcommand execution is failed %w", err)
	}
	checkPRResultStatus := checkPRResult[0].Status
	if checkPRResultStatus != "OK" {
		errorMessage := checkPRResult[0].Message
		return fmt.Errorf("The subcommand execution is failed %v", errorMessage)
	}

	resultAsserted, ok := checkPRResult[0].Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Assertion error for results of command")
	}

	fetchingLink := resultAsserted["SelfLink"].(string)
	if !ok {
		return fmt.Errorf("Assertion error")
	}

	// #2 Add reviewer to the repo: scanio integration-vcs --vcs bitbucket --action addRoleToPR --vcs-url git.com --namespace TEST --repository test --pull-request-id ID --login --role REVIWER
	addReviewerResult, err := executeCommand("integration-vcs",
		"--vcs", allDecisionMakerOptions.VCSPlugName,
		"--action", "addRoleToPR",
		"--vcs-url", allDecisionMakerOptions.VCSURL,
		"--namespace", allDecisionMakerOptions.Namespace,
		"--repository", allDecisionMakerOptions.Repository,
		"--pull-request-id", fmt.Sprintf("%v", allDecisionMakerOptions.PullRequestId),
		"--login", allDecisionMakerOptions.Login,
		"--role", "REVIWER",
	)
	if err != nil {
		return fmt.Errorf("The subcommand execution is failed %w", err)
	}
	addReviewerResultStatus := addReviewerResult[0].Status
	if addReviewerResultStatus != "OK" {
		errorMessage := addReviewerResult[0].Message
		return fmt.Errorf("The subcommand execution is failed %v", errorMessage)
	}

	//????? #3 Block the PR: scanio integration-vcs --vcs bitbucket --action setStatusOfPR --vcs-url git.com --namespace TEST --repository test --pull-request-id ID --login LOGIN --status UNAPPROVED
	changingStatusResult, err := executeCommand("integration-vcs",
		"--vcs", allDecisionMakerOptions.VCSPlugName,
		"--action", "setStatusOfPR",
		"--vcs-url", allDecisionMakerOptions.VCSURL,
		"--namespace", allDecisionMakerOptions.Namespace,
		"--repository", allDecisionMakerOptions.Repository,
		"--pull-request-id", fmt.Sprintf("%v", allDecisionMakerOptions.PullRequestId),
		"--login", allDecisionMakerOptions.Login,
		"--status", "UNAPPROVED",
	)
	if err != nil {
		return fmt.Errorf("The subcommand execution is failed %w", err)
	}
	changingStatusResultStatus := changingStatusResult[0].Status
	if changingStatusResultStatus != "OK" {
		errorMessage := changingStatusResult[0].Message
		return fmt.Errorf("The subcommand execution is failed %v", errorMessage)
	}

	// #4 Add a comment to the PR: scanio integration-vcs --vcs bitbucket --action addComment --vcs-url git.com --namespace TEST --repository test --comment "Test text"
	commentingPRResult, err := executeCommand("integration-vcs",
		"--vcs", allDecisionMakerOptions.VCSPlugName,
		"--action", "addComment",
		"--vcs-url", allDecisionMakerOptions.VCSURL,
		"--namespace", allDecisionMakerOptions.Namespace,
		"--repository", allDecisionMakerOptions.Repository,
		"--pull-request-id", fmt.Sprintf("%v", allDecisionMakerOptions.PullRequestId),
		"--comment", "Started",
	)
	if err != nil {
		return fmt.Errorf("The subcommand execution is failed %w", err)
	}
	commentingPRResultStatus := commentingPRResult[0].Status
	if commentingPRResultStatus != "OK" {
		errorMessage := commentingPRResult[0].Message
		return fmt.Errorf("The subcommand execution is failed %v", errorMessage)
	}

	// #5 Fetch repo: scanio integration-vcs --vcs bitbucket --action checkPR --vcs-url git.com --namespace TEST --repository test --pull-request-id 9
	fetchingResult, err := executeCommand("fetch", "--vcs", "bitbucket", "--auth-type", "ssh-agent", fetchingLink)
	if err != nil {
		return fmt.Errorf("The subcommand execution is failed %w", err)
	}
	fetchingResultStatus := fetchingResult[0].Status
	if fetchingResultStatus != "OK" {
		errorMessage := fetchingResult[0].Message
		return fmt.Errorf("The subcommand execution is failed %v", errorMessage)
	}

	resultAsserted, ok = fetchingResult[0].Args.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Assertion error for results of command")
	}

	scaningFolder := resultAsserted["TargetFolder"].(string)
	if !ok {
		return fmt.Errorf("Assertion error")
	}

	// #3.1 Scan code: scanio analyse --scanner semgrep --format sarif /[scanio]/test
	scanningResult, err := executeCommand("analyse",
		"--scanner", "semgrep",
		"--format", "sarif",
		scaningFolder,
	)
	if err != nil {
		return fmt.Errorf("The subcommand execution is failed %w", err)
	}
	scanningResultStatus := scanningResult[0].Status
	if scanningResultStatus != "OK" {
		errorMessage := scanningResult[0].Message
		return fmt.Errorf("The subcommand execution is failed %v", errorMessage)
	}

	// 	// #3.2 Scan code: scanio analyse --scanner trufflehog3 -c /scanio-rules/trufflehog-rules/rules.yaml /[scanio]/test -- --no-history
	// 	outputBytes, err = executeCommand("analyse",
	// 		"--scanner", "trufflehog3",
	// 		"--format", "json",
	// 		"--config", "",
	// 		scanPath,
	// 		"--", "--no-history",
	// 	)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if err := json.Unmarshal(outputBytes, &resultOfSubcommandList); err != nil {
	// 		return fmt.Errorf("Failed to parse integration-vcs output: %w", err)
	// 	}
	// 	asserted, ok = resultOfSubcommandList[0].Args.(map[string]interface{})
	// 	fmt.Print(asserted)
	// }

	// Parsing reports
	// Comment PR
	// Unblock if ok
	// Handler if something happend
	//common go handler for tasks and wrtiting comments if something happend
	//"git..com/sec/automation/pkg/jenkins"
	// 	buildLink := jenkins.BuildLinkMarkdown(jenkins.JobScanPullRequest, os.Getenv("BUILD_NUMBER"))

	return nil
}

var handlerCmd = &cobra.Command{
	Use:                   "decision-maker --scenario SCENARIO_NAME",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example:               execExampleDecisionMaker,
	Short:                 "",
	Long:                  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := shared.NewLogger("core-analyze-scanner")

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
			if err := scanioHandler(logger); err != nil {
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
