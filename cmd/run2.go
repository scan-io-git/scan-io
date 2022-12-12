/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
)

type Run2Options struct {
	AuthType          string
	InputFile         string
	Repositories      []string
	RmExts            string
	ScannerPluginName string
	SSHKey            string
	VCSPlugName       string
	ReportFormat      string
	Config            string
	AdditionalArgs    []string
	NoFetch           bool
	NoScan            bool
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

	for _, scanArgs := range scanArgsList {
		err := run2scan(scanArgs)
		if err != nil {
			return err
		}
	}

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

	for _, fetchArgs := range fetchArgsList {
		err := run2fetch(fetchArgs)
		if err != nil {
			return err
		}
	}

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

var run2Cmd = &cobra.Command{
	Use:   "run2",
	Short: "Better version of 'run'",
	// Long: `
	// `,
	RunE: func(cmd *cobra.Command, args []string) error {
		var repos []shared.RepositoryParams
		if len(allRun2Options.InputFile) > 0 {
			reposFromFile, err := utils.ReadReposFile2(allRun2Options.InputFile)
			if err != nil {
				return err
			}
			repos = reposFromFile
		}
		for _, repoURL := range allRun2Options.Repositories {
			repoParams, err := convertRawRepoURLToRepoParams(repoURL)
			if err != nil {
				return err
			}
			repos = append(repos, *repoParams)
		}

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
	},
}

func init() {
	rootCmd.AddCommand(run2Cmd)

	run2Cmd.Flags().StringVar(&allRun2Options.AuthType, "auth-type", "", "Type of authentication: 'http', 'ssh-agent' or 'ssh-key'")
	run2Cmd.Flags().StringVarP(&allRun2Options.InputFile, "input", "f", "", "file with list of repos. Results of there repos will be uploaded")
	run2Cmd.Flags().StringSliceVar(&allRun2Options.Repositories, "repos", []string{}, "list of repos to fetch - full path format=")
	run2Cmd.Flags().StringVar(&allRun2Options.RmExts, "rm-ext", "csv,png,ipynb,txt,md,mp4,zip,gif,gz,jpg,jpeg,cache,tar,svg,bin,lock,exe", "Files with extention to remove automatically after checkout")
	run2Cmd.Flags().StringVar(&allRun2Options.ScannerPluginName, "scanner", "semgrep", "scanner plugin name")
	run2Cmd.Flags().StringVar(&allRun2Options.SSHKey, "ssh-key", "", "Path to ssh key")
	run2Cmd.Flags().StringVar(&allRun2Options.VCSPlugName, "vcs", "", "vcs plugin name")
	run2Cmd.Flags().StringVarP(&allRun2Options.Config, "config", "c", "auto", "")
	run2Cmd.Flags().StringVarP(&allRun2Options.ReportFormat, "format", "o", "", "") //doesn't have default for "Uses ASCII output if no format specified"
	run2Cmd.Flags().StringSliceVar(&allRun2Options.AdditionalArgs, "args", []string{}, "additional commands for scanner which are will be added to a scanner call. Format in quots with commas withous spaces.")
	run2Cmd.Flags().BoolVar(&allRun2Options.NoFetch, "no-fetch", false, "skip fetch stage")
	run2Cmd.Flags().BoolVar(&allRun2Options.NoScan, "no-scan", false, "skip scan stage")
}
