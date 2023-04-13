package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	utils "github.com/scan-io-git/scan-io/internal/utils"
	"github.com/scan-io-git/scan-io/pkg/shared"
)

type Scanner struct {
	additionalArgs    []string
	config            string
	scannerPluginName string
	reportFormat      string
	logger            hclog.Logger
	jobs              int
}

func New(scannerPluginName string, jobs int, config string, reportFormat string, additionalArgs []string, logger hclog.Logger) Scanner {
	return Scanner{
		additionalArgs:    additionalArgs,
		config:            config,
		scannerPluginName: scannerPluginName,
		reportFormat:      reportFormat,
		logger:            logger,
		jobs:              jobs,
	}
}

func (s Scanner) PrepScanArgs(repos []shared.RepositoryParams, path string) ([]shared.ScannerScanRequest, error) {
	var scanArgs []shared.ScannerScanRequest
	var targetFolder string
	var resultsPath string

	// make dinamic extension name, based on output format
	reportExt := "raw"
	if len(s.reportFormat) > 0 {
		reportExt = s.reportFormat
	}
	if len(path) != 0 {
		// in the case with a manual path the result will be written to the same folder
		targetFolder = path
		resultsPath = filepath.Join(path, fmt.Sprintf("%s.%s", s.scannerPluginName, reportExt))

		scanArgs = append(scanArgs, shared.ScannerScanRequest{
			RepoPath:       targetFolder,
			ResultsPath:    resultsPath,
			ConfigPath:     s.config,
			AdditionalArgs: s.additionalArgs,
			ReportFormat:   s.reportFormat,
		})
	} else {
		for _, repo := range repos {
			domain, err := utils.GetDomain(repo.SshLink)
			if err != nil {
				domain, err = utils.GetDomain(repo.HttpLink)
				if err != nil {
					return nil, err
				}
			}

			resultsFolderPath := filepath.Join(shared.GetResultsHome(), strings.ToLower(domain), filepath.Join(strings.ToLower(repo.Namespace), strings.ToLower(repo.RepoName)))
			// ensure that folder for results exists, some scanners don't create it themselves and just exit with an error
			if err := os.MkdirAll(resultsFolderPath, os.ModePerm); err != nil {
				return nil, err
			}

			targetFolder = shared.GetRepoPath(strings.ToLower(domain), filepath.Join(strings.ToLower(repo.Namespace), strings.ToLower(repo.RepoName)))
			resultsPath = filepath.Join(resultsFolderPath, fmt.Sprintf("%s.%s", s.scannerPluginName, reportExt))

			scanArgs = append(scanArgs, shared.ScannerScanRequest{
				RepoPath:       targetFolder,
				ResultsPath:    resultsPath,
				ConfigPath:     s.config,
				AdditionalArgs: s.additionalArgs,
				ReportFormat:   s.reportFormat,
			})
		}
	}

	return scanArgs, nil
}

func (s Scanner) scanRepo(scanArg shared.ScannerScanRequest) error {

	shared.WithPlugin("plugin-scanner", shared.PluginTypeScanner, s.scannerPluginName, func(raw interface{}) {
		scanner := raw.(shared.Scanner)
		err := scanner.Scan(scanArg)
		if err != nil {
			s.logger.Error("Scanner plugin is failed")
		}
	})

	return nil
}

func (s Scanner) ScanRepos(scanArgs []shared.ScannerScanRequest) error {

	s.logger.Info("Scan starting", "total", len(scanArgs), "goroutines", s.jobs)

	values := make([]interface{}, len(scanArgs))
	for i := range scanArgs {
		values[i] = scanArgs[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(s.jobs, values, func(i int, value interface{}) {
		scanArg := value.(shared.ScannerScanRequest)
		s.logger.Info("Goroutine started", "#", i+1, "args", scanArg)

		err := s.scanRepo(scanArg)
		if err != nil {
			s.logger.Error("scanners's scanRepo() failed", "err", err)
		}
	})

	return nil
}
