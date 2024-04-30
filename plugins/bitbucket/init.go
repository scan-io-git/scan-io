package main

import (
	"fmt"
	"os"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

// Init function for checking an environment
func (g *VCSBitbucket) init(command string, authType string) (shared.EvnVariables, error) {
	var variables shared.EvnVariables
	variables.Username = os.Getenv("SCANIO_BITBUCKET_USERNAME")
	variables.Token = os.Getenv("SCANIO_BITBUCKET_TOKEN")
	variables.SshKeyPassword = os.Getenv("SCANIO_BITBUCKET_SSH_KEY_PASSWORD")

	if command == "list" && ((len(variables.Username) == 0) || (len(variables.Token) == 0)) {
		err := fmt.Errorf("SCANIO_BITBUCKET_USERNAME or SCANIO_BITBUCKET_TOKEN is not provided in an environment.")
		g.logger.Error("An insufficiently configured environment", "error", err)
		return variables, err
	}

	if command == "fetch" {
		if len(variables.SshKeyPassword) == 0 && authType == "ssh-key" {
			g.logger.Warn("SCANIO_BITBUCKET_SSH_KEY_PASSOWRD is empty or not provided.")
		}

		if authType == "http" && ((len(variables.Username) == 0) || (len(variables.Token) == 0)) {
			err := fmt.Errorf("SCANIO_BITBUCKET_USERNAME or SCANIO_BITBUCKET_TOKEN is not provided in an environment.")
			g.logger.Error("An insufficiently configured environment", "error", err)
			return variables, err
		}
	}
	return variables, nil
}
