package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"

	utils "github.com/scan-io-git/scan-io/internal/utils"
)

// Scanner represents the basic configuration and behavior of a scanner.
type Scanner struct {
	pluginName     string       // Name of the scanner plugin to use
	configPath     string       // Path to the configuration file for the scanner
	reportFormat   string       // Format of the report to generate (e.g., JSON, Sarif)
	additionalArgs []string     // Additional arguments for the scanner
	concurrentJobs int          // Number of concurrent jobs to run
	logger         hclog.Logger // Logger for logging messages and errors
}

// New creates a new Scanner instance with the provided configuration.
func New(pluginName, configPath, reportFormat string, additionalArgs []string, concurrentJobs int, logger hclog.Logger) *Scanner {
	return &Scanner{
		pluginName:     pluginName,
		configPath:     configPath,
		reportFormat:   reportFormat,
		additionalArgs: additionalArgs,
		concurrentJobs: concurrentJobs,
		logger:         logger,
	}
}

// PrepareScanArgs prepares the arguments needed for the scan operation with the provided configuration.
func (s *Scanner) PrepareScanArgs(cfg *config.Config, repos []shared.RepositoryParams, targetPath, outputPath string) ([]shared.ScannerScanRequest, error) {
	var scanArgs []shared.ScannerScanRequest

	// Determine report extension based on the format
	reportExt := "raw"
	if s.reportFormat != "" {
		reportExt = s.reportFormat
	}

	// Determine the name template based on the CI mode
	nameTemplate := s.generateNameTemplate(cfg, reportExt)

	// Handle single target path scenario
	if targetPath != "" {
		resultsFile, err := s.determineResultsFilePath(targetPath, outputPath, nameTemplate)
		if err != nil {
			return nil, err
		}

		scanArgs = append(scanArgs, shared.ScannerScanRequest{
			TargetPath:     targetPath,
			ResultsPath:    resultsFile,
			ConfigPath:     s.configPath,
			ReportFormat:   s.reportFormat,
			AdditionalArgs: s.additionalArgs,
		})
	} else {
		// Handle multiple repositories scenario
		for _, repo := range repos {
			scanArg, err := s.prepareRepoScanArg(cfg, repo, nameTemplate)
			if err != nil {
				return nil, err
			}
			scanArgs = append(scanArgs, scanArg)
		}
	}

	return scanArgs, nil
}

// prepareRepoScanArg prepares the scan arguments for a repository.
func (s *Scanner) prepareRepoScanArg(cfg *config.Config, repo shared.RepositoryParams, nameTemplate string) (shared.ScannerScanRequest, error) {
	domain, err := utils.GetDomain(repo.SshLink)
	if err != nil {
		domain, err = utils.GetDomain(repo.HttpLink)
		if err != nil {
			return shared.ScannerScanRequest{}, err
		}
	}

	resultsFolderPath := filepath.Join(config.GetScanioResultsHome(cfg), strings.ToLower(domain), strings.ToLower(repo.Namespace), strings.ToLower(repo.RepoName))
	targetPath := config.GetRepositoryPath(cfg, domain, filepath.Join(repo.Namespace, repo.RepoName))
	resultsFile := filepath.Join(resultsFolderPath, nameTemplate)

	if err := files.CreateFolderIfNotExists(resultsFolderPath); err != nil {
		return shared.ScannerScanRequest{}, fmt.Errorf("failed to create results folder '%s': %w", resultsFolderPath, err)
	}

	return shared.ScannerScanRequest{
		TargetPath:     targetPath,
		ResultsPath:    resultsFile,
		ConfigPath:     s.configPath,
		ReportFormat:   s.reportFormat,
		AdditionalArgs: s.additionalArgs,
	}, nil
}

// determineResultsFilePath determines the results file path based on target and output paths.
func (s *Scanner) determineResultsFilePath(targetPath, outputPath, nameTemplate string) (string, error) {
	if outputPath != "" {
		return s.handleOutputPath(outputPath, nameTemplate)
	}
	return s.handleOutputPath(targetPath, nameTemplate)
}

// handleOutputPath handles the output path, creating directories as necessary.
func (s *Scanner) handleOutputPath(path, nameTemplate string) (string, error) {
	var resultsFile, resultsFolder string

	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo.IsDir() {
		// It's a directory
		resultsFolder = path
		resultsFile = filepath.Join(path, nameTemplate)
	} else {
		// It's a file or doesn't exist
		ext := filepath.Ext(path)
		if ext == "" {
			// No extension, treat as directory
			resultsFolder = path
			resultsFile = filepath.Join(path, nameTemplate)
		} else {
			// Has extension, treat as file
			resultsFolder = filepath.Dir(path)
			resultsFile = path
		}
	}

	if err := files.CreateFolderIfNotExists(resultsFolder); err != nil {
		return "", fmt.Errorf("failed to create results folder '%s': %w", resultsFolder, err)
	}

	return resultsFile, nil
}

// generateNameTemplate generates a name template for the results file based on the CI mode.
func (s *Scanner) generateNameTemplate(cfg *config.Config, reportExt string) string {
	nameTemplate := fmt.Sprintf("scanio-report-%s.%s", s.pluginName, reportExt)
	if !config.IsCI(cfg) {
		startTime := time.Now().UTC().Format(time.RFC3339)
		nameTemplate = fmt.Sprintf("scanio-report-%s-%s.%s", s.pluginName, startTime, reportExt)
	}
	return nameTemplate
}

func (s Scanner) scanRepo(cfg *config.Config, scanArg shared.ScannerScanRequest) (shared.ScannerScanResponse, error) {
	var (
		result shared.ScannerScanResponse
		err    error
	)

	err = shared.WithPlugin(cfg, "plugin-scanner", shared.PluginTypeScanner, s.pluginName, func(raw interface{}) error {
		scanner := raw.(shared.Scanner)
		result, err = scanner.Scan(scanArg)
		if err != nil {
			s.logger.Error("Scanner plugin is failed")
			return err
		}
		return nil
	})

	return result, err
}

func (s Scanner) ScanRepos(cfg *config.Config, scanArgs []shared.ScannerScanRequest) shared.GenericLaunchesResult {

	s.logger.Info("Scan starting", "total", len(scanArgs), "goroutines", s.concurrentJobs)

	var results shared.GenericLaunchesResult
	resultsChannel := make(chan shared.GenericResult, len(scanArgs))
	values := make([]interface{}, len(scanArgs))
	for i := range scanArgs {
		values[i] = scanArgs[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(s.concurrentJobs, values, func(i int, value interface{}) {
		scanArg := value.(shared.ScannerScanRequest)
		s.logger.Info("Goroutine started", "#", i+1, "args", scanArg)

		result, err := s.scanRepo(cfg, scanArg)
		if err != nil {
			s.logger.Error("scanners's scanRepo() failed", "err", err)
			resultAnalyse := shared.GenericResult{Args: scanArg, Result: result, Status: "FAILED", Message: err.Error()}
			resultsChannel <- resultAnalyse
		} else {
			resultAnalyse := shared.GenericResult{Args: scanArg, Result: result, Status: "OK", Message: ""}
			resultsChannel <- resultAnalyse
		}
	})

	close(resultsChannel)
	for result := range resultsChannel {
		results.Launches = append(results.Launches, result)
	}
	return results
}
