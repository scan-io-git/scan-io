package common

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

func ReadReposFile(inputFile string) ([]shared.RepositoryParams, error) {
	var file shared.GenericLaunchesResult

	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if len(file.Launches) == 0 {
		return nil, fmt.Errorf("no data in file")
	}

	var repos []shared.RepositoryParams
	for _, launch := range file.Launches {
		resultBytes, err := json.Marshal(launch.Result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal launch.Result: %w", err)
		}

		var namespaces []shared.NamespaceParams
		if err := json.Unmarshal(resultBytes, &namespaces); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result into []NamespaceParams: %w", err)
		}
		for _, ns := range namespaces {
			repos = append(repos, ns.Repositories...)
		}
	}

	return repos, nil
}

func GetDomain(repositoryURL string) (string, error) {
	if strings.HasPrefix(repositoryURL, "git@") && strings.HasSuffix(repositoryURL, ".git") {
		u := repositoryURL[4 : len(repositoryURL)-4]
		splitter := "/"
		if strings.Contains(u, ":") {
			splitter = ":"
		}
		return strings.Split(u, splitter)[0], nil
	}

	parsedUrl, err := url.Parse(repositoryURL)
	if err != nil {
		return "", fmt.Errorf("error during parsing repositoryURL %q: %w", repositoryURL, err)
	}

	parts := strings.Split(parsedUrl.Host, ":")
	switch len(parts) {
	case 1:
		fallthrough
	case 2:
		return parts[0], nil
	default:
		return "", fmt.Errorf("unable to get domain from %q", parsedUrl.Host)
	}
}

func GetPath(repositoryURL string) (string, error) {
	if strings.HasPrefix(repositoryURL, "git@") && strings.HasSuffix(repositoryURL, ".git") {
		url := strings.TrimPrefix(repositoryURL, "git@")
		url = strings.TrimSuffix(url, ".git")
		parts := strings.Split(url, ":")
		if len(parts) != 2 {
			return "", fmt.Errorf("unknown format of url: %q", repositoryURL)
		}
		return parts[1], nil
	}

	parsedUrl, err := url.Parse(repositoryURL)
	if err != nil {
		return "", fmt.Errorf("error during parsing repositoryURL %q: %w", repositoryURL, err)
	}

	path := parsedUrl.Path
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, ".git")
	return path, nil
}

func SplitPathOnNamespaceAndRepoName(path string) (string, string) {
	pathParts := strings.Split(path, "/")
	namespace := strings.Join(pathParts[:len(pathParts)-1], "/")
	repoName := pathParts[len(pathParts)-1]
	return namespace, repoName
}
