package common

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

func ReadReposFile(inputFile string) ([]string, error) {
	readFile, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer readFile.Close()

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	lines := []string{}
	for fileScanner.Scan() {
		lines = append(lines, fileScanner.Text())
	}

	return lines, nil
}

func ReadReposFile2(inputFile string) ([]shared.RepositoryParams, error) {
	var wholeFile shared.ListFuncResult
	var result []shared.RepositoryParams

	readFile, err := os.Open(inputFile)
	if err != nil {
		return result, err
	}
	defer readFile.Close()

	byteValue, _ := ioutil.ReadAll(readFile)
	err = json.Unmarshal(byteValue, &wholeFile)
	if err != nil {
		return result, err
	}
	return wholeFile.Result, nil
}

func GetDomain(repositoryURL string) (string, error) {

	if strings.HasPrefix(repositoryURL, "git@") && strings.HasSuffix(repositoryURL, ".git") {
		return strings.Split(repositoryURL[4:], ":")[0], nil
	}

	parsedUrl, err := url.Parse(repositoryURL)
	if err != nil {
		return "", fmt.Errorf("error during parsing repositoryURL '%s': %w", repositoryURL, err)
	}

	parts := strings.Split(parsedUrl.Host, ":")
	switch len(parts) {
	case 1:
		fallthrough
	case 2:
		return parts[0], nil
	default:
		return "", fmt.Errorf("unable to get domain from %s", parsedUrl.Host)
	}
}

func GetPath(repositoryURL string) (string, error) {

	if strings.HasPrefix(repositoryURL, "git@") && strings.HasSuffix(repositoryURL, ".git") {
		url := strings.TrimPrefix(repositoryURL, "git@")
		url = strings.TrimSuffix(url, ".git")
		parts := strings.Split(url, ":")
		if len(parts) != 2 {
			return "", fmt.Errorf("unknown format of url: %s", repositoryURL)
		}
		return parts[1], nil
	}

	parsedUrl, err := url.Parse(repositoryURL)
	if err != nil {
		return "", fmt.Errorf("error during parsing repositoryURL '%s': %w", repositoryURL, err)
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
