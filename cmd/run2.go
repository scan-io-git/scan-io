/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/scan-io-git/scan-io/internal/fetcher"
	"github.com/scan-io-git/scan-io/internal/scanner"
	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

type Run2Options struct {
	AuthType          string
	InputFile         string
	Repositories      []string
	RmExts            string
	Runtime           string
	ScannerPluginName string
	SSHKey            string
	Branch            string
	VCSPlugName       string
	ReportFormat      string
	Config            string
	AdditionalArgs    []string
	NoFetch           bool
	NoScan            bool
	Jobs              int
}

var allRun2Options Run2Options

func run2analyzeRepos(repos []shared.RepositoryParams) error {

	logger := logger.NewLogger(AppConfig, "core-run2-scanner")
	s := scanner.New(allRun2Options.ScannerPluginName, allRun2Options.Jobs, allRun2Options.Config, allRun2Options.ReportFormat, allRun2Options.AdditionalArgs, logger)

	scanArgs, err := s.PrepScanArgs(AppConfig, repos, "", "")
	if err != nil {
		return err
	}

	_ = s.ScanRepos(AppConfig, scanArgs)

	return nil
}

func run2fetchRepos(repos []shared.RepositoryParams) error {

	logger := logger.NewLogger(AppConfig, "core-run2-fetcher")
	f := fetcher.New(allRun2Options.AuthType, allRun2Options.SSHKey, allRun2Options.Jobs, allRun2Options.Branch, allRun2Options.VCSPlugName, strings.Split(allRun2Options.RmExts, ","), logger)

	fetchArgs, err := f.PrepFetchArgs(AppConfig, logger, repos)
	if err != nil {
		return err
	}

	_ = f.FetchRepos(AppConfig, fetchArgs)

	return nil
}

func convertRawRepoURLToRepoParams(repoURL string) (*shared.RepositoryParams, error) {
	path, err := utils.GetPath(repoURL)
	if err != nil {
		return nil, err
	}
	namespace, repoName := utils.SplitPathOnNamespaceAndRepoName(path)
	return &shared.RepositoryParams{
		Namespace: namespace,
		RepoName:  repoName,
		HttpLink:  repoURL,
		SshLink:   repoURL,
	}, nil
}

func prepRepos() ([]shared.RepositoryParams, error) {
	var repos []shared.RepositoryParams

	if len(allRun2Options.InputFile) > 0 {
		reposFromFile, err := utils.ReadReposFile2(allRun2Options.InputFile)
		if err != nil {
			return nil, err
		}
		repos = reposFromFile
	}

	for _, repoURL := range allRun2Options.Repositories {
		repoParams, err := convertRawRepoURLToRepoParams(repoURL)
		if err != nil {
			return nil, err
		}
		repos = append(repos, *repoParams)
	}

	return repos, nil
}

func run2Locally(repos []shared.RepositoryParams) error {

	if !allRun2Options.NoFetch {
		err := run2fetchRepos(repos)
		if err != nil {
			return err
		}
	}

	if !allRun2Options.NoScan {
		err := run2analyzeRepos(repos)
		if err != nil {
			return err
		}
	}

	return nil
}

func run2WithHelm(repos []shared.RepositoryParams) error {
	logger := logger.NewLogger(AppConfig, "core-run2")

	values := make([]interface{}, len(repos))
	for i := range repos {
		values[i] = repos[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(allRun2Options.Jobs, values, func(i int, value interface{}) {
		repo := value.(shared.RepositoryParams)

		jobID := uuid.New()

		repoURL := repo.HttpLink
		if allRun2Options.AuthType != "http" {
			repoURL = repo.SshLink
		}

		logger.Info("run2WithHelm's goroutine started", "#", i+1, "repo", repoURL)

		remoteCommandArgs := []string{
			"scanio", "run2",
			"--auth-type", allRun2Options.AuthType,
			"--ssh-key", allRun2Options.SSHKey,
			"--vcs", allRun2Options.VCSPlugName,
			"--scanner", allRun2Options.ScannerPluginName,
			"--repos", repoURL,
		}
		jobCommand := fmt.Sprintf("command={%s}", strings.Join(remoteCommandArgs, ","))

		jobChartPath := DEFAULT_JOB_HELM_CHART_PATH
		if path := os.Getenv("JOB_HELM_CHART_PATH"); path != "" {
			jobChartPath = path
		}

		cmd := exec.Command("helm", "install", jobID.String(), jobChartPath,
			"--set", jobCommand,
			// "--set", "image.repository=scanio",
			// "--set", "image.tag=latest",
			"--set", fmt.Sprintf("suffix=%s", jobID.String()),
		)
		if err := cmd.Run(); err != nil {
			// logger.Debug("helm install error", "err", err)
			// log.Fatal(err)
			panic(err)
			// return err
		}

		jobsClient := getNewJobsClient()

		jobName := fmt.Sprintf("scanio-job-%s", jobID.String())

		// logger.Info("Waiting the job", "jobName", jobName)
		for {
			job, err := jobsClient.Get(context.Background(), jobName, metav1.GetOptions{})
			if err != nil {
				panic(err)
			}
			if job.Status.Succeeded > 0 || job.Status.Failed == *job.Spec.BackoffLimit+1 {
				break
			}
		}

		logger.Info("run2WithHelm's goroutine ending", "#", i+1, "repo", repoURL)

		cmd = exec.Command("helm", "uninstall", jobID.String())
		if err := cmd.Run(); err != nil {
			// log.Fatal(err)
			panic(err)
			// return
		}
	})

	return nil
}

var run2Cmd = &cobra.Command{
	Use:   "run2",
	Short: "[EXPERIMENTAL] Better version of 'run'",
	Long: `
		run2 command is a combination of fetch and analyze commands.
		Actively used for remote runtime (--runtime helm).
		But you can use it locally too (--runtime local).
	`,
	RunE: func(cmd *cobra.Command, args []string) error {

		repos, err := prepRepos()
		if err != nil {
			return err
		}

		if allRun2Options.Runtime == "helm" {
			return run2WithHelm(repos)
		} else if allRun2Options.Runtime == "local" {
			return run2Locally(repos)
		} else {
			return fmt.Errorf("unknown runtime")
		}
	},
}

func init() {
	rootCmd.AddCommand(run2Cmd)

	run2Cmd.Flags().StringVar(&allRun2Options.AuthType, "auth-type", "", "Type of authentication: 'http', 'ssh-agent' or 'ssh-key'")
	run2Cmd.Flags().StringVarP(&allRun2Options.InputFile, "input", "i", "", "file with list of repos. Results of there repos will be uploaded")
	run2Cmd.Flags().StringSliceVar(&allRun2Options.Repositories, "repos", []string{}, "list of repos to fetch - full path format")
	run2Cmd.Flags().StringVar(&allRun2Options.RmExts, "rm-ext", "csv,png,ipynb,txt,md,mp4,zip,gif,gz,jpg,jpeg,cache,tar,svg,bin,lock,exe", "Files with extention to remove automatically after checkout")
	run2Cmd.Flags().StringVar(&allRun2Options.ScannerPluginName, "scanner", "semgrep", "scanner plugin name")
	run2Cmd.Flags().StringVar(&allRun2Options.SSHKey, "ssh-key", "", "Path to ssh key")
	run2Cmd.Flags().StringVar(&allRun2Options.VCSPlugName, "vcs", "", "vcs plugin name")
	run2Cmd.Flags().StringVarP(&allRun2Options.Config, "config", "c", "auto", "")
	run2Cmd.Flags().StringVarP(&allRun2Options.ReportFormat, "format", "o", "", "") //doesn't have default for "Uses ASCII output if no format specified"
	run2Cmd.Flags().StringSliceVar(&allRun2Options.AdditionalArgs, "args", []string{}, "additional commands for scanner which are will be added to a scanner call. Format in quots with commas withous spaces.")
	run2Cmd.Flags().BoolVar(&allRun2Options.NoFetch, "no-fetch", false, "skip fetch stage")
	run2Cmd.Flags().BoolVar(&allRun2Options.NoScan, "no-scan", false, "skip scan stage")
	run2Cmd.Flags().StringVar(&allRun2Options.Runtime, "runtime", "local", "runtime 'local' or 'helm'")
	run2Cmd.Flags().IntVarP(&allRun2Options.Jobs, "jobs", "j", 1, "jobs to run in parallel")
}
