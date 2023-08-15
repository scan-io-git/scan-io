package common

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

type HTTPClient struct {
	Client *http.Client
}

func NewHTTPClient(proxyUrl string, skipVerification bool) *HTTPClient {
	// Disable or not SSL/TLS verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerification},
	}
	// Create a proxy URL
	proxyURL, err := url.Parse(proxyUrl)
	if err != nil {
		fmt.Println("Error parsing proxy URL:", err)
		return nil
	}

	// Set the proxy for the transport
	tr.Proxy = http.ProxyURL(proxyURL)

	// Create an HTTP client with the custom transport
	client := &http.Client{Transport: tr}

	return &HTTPClient{Client: client}
}

func (c *HTTPClient) DoRequest(method, url string, headers http.Header, body []byte) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	var response *http.Response
	var responseBody []byte

	if err != nil {
		return response, responseBody, err
	}

	// Add custom headers
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Send the request using the shared client
	response, err = c.Client.Do(req)
	if err != nil {
		return response, responseBody, err
	}

	responseBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return response, responseBody, err
	}

	defer response.Body.Close()

	return response, responseBody, nil
}

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
		u := repositoryURL[4 : len(repositoryURL)-4]
		splitter := "/"
		if strings.Contains(u, ":") {
			splitter = ":"
		}
		return strings.Split(u, splitter)[0], nil
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

func FindByExtAndRemove(root string, exts []string) {
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		ext := filepath.Ext(d.Name())
		match := false
		for _, rmExt := range exts {
			if fmt.Sprintf(".%s", rmExt) == ext {
				match = true
				break
			}
		}
		if !match {
			return nil
		}
		e = os.Remove(s)
		if e != nil {
			return e
		}
		return nil
	})
}
