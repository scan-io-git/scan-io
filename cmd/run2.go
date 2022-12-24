/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Run2Options struct {
	AuthType          string
	InputFile         string
	Repositories      []string
	RmExts            string
	Runtime           string
	ScannerPluginName string
	SSHKey            string
	VCSPlugName       string
	ReportFormat      string
	Config            string
	AdditionalArgs    []string
	NoFetch           bool
	NoScan            bool
	Jobs              int
}

var allRun2Options Run2Options

func prepScanArgs(repo shared.RepositoryParams) (*shared.ScannerScanRequest, error) {
	var cloneURL string
	if allRun2Options.AuthType == "http" {
		cloneURL = repo.HttpLink
	} else {
		cloneURL = repo.SshLink
	}

	domain, err := utils.GetDomain(cloneURL)
	if err != nil {
		return nil, err
	}

	repoFolder := shared.GetRepoPath(domain, filepath.Join(repo.Namespace, repo.RepoName))
	resultsFolderPath := filepath.Join(shared.GetResultsHome(), domain, filepath.Join(repo.Namespace, repo.RepoName))

	if err := os.MkdirAll(resultsFolderPath, os.ModePerm); err != nil {
		return nil, err
		// log.Fatal(err)
	}

	reportExt := "raw"
	if len(allRun2Options.ReportFormat) > 0 {
		reportExt = allRun2Options.ReportFormat
	}
	resultsPath := filepath.Join(resultsFolderPath, fmt.Sprintf("%s.%s", allRun2Options.ScannerPluginName, reportExt))
	return &shared.ScannerScanRequest{
		RepoPath:       repoFolder,
		ResultsPath:    resultsPath,
		ConfigPath:     allRun2Options.Config,
		AdditionalArgs: allRun2Options.AdditionalArgs,
		ReportFormat:   allRun2Options.ReportFormat,
	}, nil
}

func run2scan(scanArgs shared.ScannerScanRequest) error {

	shared.WithPlugin("plugin-scanner", shared.PluginTypeScanner, allRun2Options.ScannerPluginName, func(raw interface{}) {
		scanName := raw.(shared.Scanner)
		err := scanName.Scan(scanArgs)
		if err != nil {
			logger := shared.NewLogger("core")
			logger.Warn("problem on scan", "error", err)
		}
	})

	return nil
}

func run2analyzeRepos(repos []shared.RepositoryParams) error {
	var scanArgsList []shared.ScannerScanRequest

	for _, repo := range repos {
		scanArgs, err := prepScanArgs(repo)
		if err != nil {
			return err
		}
		scanArgsList = append(scanArgsList, *scanArgs)
	}

	logger := shared.NewLogger("core-run2")

	values := make([]interface{}, len(scanArgsList))
	for i := range scanArgsList {
		values[i] = scanArgsList[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(allRun2Options.Jobs, values, func(i int, value interface{}) {
		scanArgs := value.(shared.ScannerScanRequest)

		err := run2scan(scanArgs)
		if err != nil {
			logger.Error("run2scan error", "err", err)
			// return err
		}
	})

	return nil
}

func run2fetch(fetchArgs shared.VCSFetchRequest) error {

	shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, allRun2Options.VCSPlugName, func(raw interface{}) {
		vcsName := raw.(shared.VCS)
		err := vcsName.Fetch(fetchArgs)
		if err == nil {
			findByExtAndRemove(fetchArgs.TargetFolder, strings.Split(allRun2Options.RmExts, ","))
		}
	})

	return nil
}

func prepFetchArgs(repo shared.RepositoryParams) (*shared.VCSFetchRequest, error) {
	var cloneURL string
	if allRun2Options.AuthType == "http" {
		cloneURL = repo.HttpLink
	} else {
		cloneURL = repo.SshLink
	}

	domain, err := utils.GetDomain(cloneURL)
	if err != nil {
		return nil, err
	}

	targetFolder := shared.GetRepoPath(domain, filepath.Join(repo.Namespace, repo.RepoName))

	return &shared.VCSFetchRequest{
		CloneURL:     cloneURL,
		AuthType:     allRun2Options.AuthType,
		SSHKey:       allRun2Options.SSHKey,
		TargetFolder: targetFolder,
	}, nil
}

func run2fetchRepos(repos []shared.RepositoryParams) error {
	var fetchArgsList []shared.VCSFetchRequest

	for _, repo := range repos {
		fetchArgs, err := prepFetchArgs(repo)
		if err != nil {
			return err
		}
		fetchArgsList = append(fetchArgsList, *fetchArgs)
	}

	logger := shared.NewLogger("core-run2")

	values := make([]interface{}, len(fetchArgsList))
	for i := range fetchArgsList {
		values[i] = fetchArgsList[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(allRun2Options.Jobs, values, func(i int, value interface{}) {
		fetchArgs := value.(shared.VCSFetchRequest)
		err := run2fetch(fetchArgs)
		if err != nil {
			logger.Error("run2fetch error", "err", err)
			// return err
		}
	})

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
	logger := shared.NewLogger("core-run2")

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
	Short: "Better version of 'run'",
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
	run2Cmd.Flags().StringVarP(&allRun2Options.InputFile, "input", "f", "", "file with list of repos. Results of there repos will be uploaded")
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