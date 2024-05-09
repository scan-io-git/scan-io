package bitbucket

// extractCloneLinks parses the clone links from the repository information to find HTTP and SSH URLs.
func ExtractCloneLinks(clones []CloneLink) (string, string) {
	var httpLink, sshLink string
	for _, clone := range clones {
		if clone.Name == "http" {
			httpLink = clone.Href
		} else if clone.Name == "ssh" {
			sshLink = clone.Href
		}
	}
	return httpLink, sshLink
}
