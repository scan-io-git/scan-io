package cmd

import (
	"bytes"
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

	Comment string
	Login   string
	Role    string
	Status  string
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

func executeCommand(command string, args ...string) ([]byte, error) {
	rootCmd.SetArgs(append([]string{command}, args...))

	var outputBuffer bytes.Buffer
	rootCmd.SetOutput(&outputBuffer)

	if err := rootCmd.Execute(); err != nil {
		return nil, fmt.Errorf("Failed to execute %s command: %w", command, err)
	}
	resultBufferContent := ReadResultBuffer()

	return resultBufferContent, nil
}

func scanioHandler(logger hclog.Logger) error {
	//Base command: scanio decision-maker --scenario scanPR --vcs bitbucket --vcs-url git.com --namespace TEST --repository test --pull-request-id ID

	//#1 CheckPR: scanio integration-vcs --vcs bitbucket --action checkPR --vcs-url git.com --namespace TEST --repository test --pull-request-id ID
	outputBytes, err := executeCommand("integration-vcs",
		"--vcs", allDecisionMakerOptions.VCSPlugName,
		"--action", "checkPR",
		"--vcs-url", allDecisionMakerOptions.VCSURL,
		"--namespace", allDecisionMakerOptions.Namespace,
		"--repository", allDecisionMakerOptions.Repository,
		"--pull-request-id", fmt.Sprintf("%v", allDecisionMakerOptions.PullRequestId),
	)
	if err != nil {
		return err
	}

	var resultOfSubcommand shared.GenericResult
	if err := json.Unmarshal(outputBytes, &resultOfSubcommand); err != nil {
		return fmt.Errorf("Failed to parse integration-vcs output: %w", err)
	}

	prParams, ok := resultOfSubcommand.Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Assertion error")
	}
	selfLink, ok := prParams["SelfLink"].(string)

	// #2 Fetch repo: scanio integration-vcs --vcs bitbucket --action checkPR --vcs-url git.com --namespace TEST --repository test --pull-request-id 9
	outputBytes, err = executeCommand("fetch", "--vcs", "bitbucket", "--auth-type", "ssh-agent", selfLink)
	// if err != nil {
	// 	return err
	// }

	// if err := json.Unmarshal(outputBytes, &resultOfSubcommand); err != nil {
	// 	return fmt.Errorf("Failed to parse integration-vcs output: %w", err)
	// }

	// #3 Scan code: scanio analyse --scanner semgrep --format sarif /[scanio]/test
	// outputBytes, err = executeCommand("analyse",
	// 	"--scanner", "semgrep",
	// 	"--format", "sarif",
	// 	"",
	// )

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
			logger.Info("Starting scanPR scenarion")
			if err := scanioHandler(logger); err != nil {
				return err
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
	// handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Action, "action", "", "the action to execute.")
	handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Namespace, "namespace", "", "the name of a specific namespace. Namespace for Gitlab is an organization, for Bitbucket_v1 is a project.")
	handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Repository, "repository", "", "the name of a specific repository.")
	handlerCmd.Flags().IntVar(&allDecisionMakerOptions.PullRequestId, "pull-request-id", 0, "the id of specific PR form the repository.")
	// handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Login, "login", "", "login for integrations. For example, add reviewer with this login to PR.")
	// handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Role, "role", "", "role for integrations. For example, add a person with specific role to PR.")
	// handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Status, "status", "", "status for integrations. For example, set a status of PR.")
	// handlerCmd.Flags().StringVar(&allDecisionMakerOptions.Comment, "comment", "", "comment for integrations. The text will be used like a comment to PR")
}
