package utils

import (
	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/mitchellh/mapstructure"
)

type BBReposLinks struct {
	Name     string
	HttpLink string
	SshLink  string
}

type BBProject struct {
	Key  string
	Link string
}

func GetProjectsResponse(r *bitbucketv1.APIResponse) ([]bitbucketv1.Project, error) {
	var m []bitbucketv1.Project
	err := mapstructure.Decode(r.Values["values"], &m)
	return m, err
}
