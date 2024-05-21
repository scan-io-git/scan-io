package bitbucket

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// ExtractCloneLinks parses the clone links from the repository information and returns the HTTP and SSH URLs.
func ExtractCloneLinks(clones []CloneLink) (httpLink, sshLink string) {
	for _, clone := range clones {
		switch clone.Name {
		case "http":
			httpLink = clone.Href
		case "ssh":
			sshLink = clone.Href
		}
	}
	return
}

// trimPRLink trims straidforwardly a pull request URL to point on a repository.
func trimPRLink(inputURL string) (string, error) {
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL to trim: %w", err)
	}

	segments := strings.Split(parsedURL.Path, "/")
	if len(segments) < 6 {
		return "", fmt.Errorf("URL path does not have enough segments to trim")
	}

	basePath := path.Join(segments[:6]...)
	parsedURL.Path = basePath

	return parsedURL.String(), nil
}
