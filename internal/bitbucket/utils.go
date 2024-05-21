package bitbucket

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
