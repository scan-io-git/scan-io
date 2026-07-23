package tohtml

import (
	"testing"

	"github.com/scan-io-git/scan-io/pkg/shared/vcsurl"
)

func TestResolvePullRequestID(t *testing.T) {
	tests := []struct {
		name string
		flag string
		vcs  vcsurl.VCSType
		env  map[string]string
		want string
	}{
		{"flag beats env", "55", vcsurl.Github, map[string]string{"GITHUB_REF": "refs/pull/123/merge"}, "55"},
		{"github ref parsed", "", vcsurl.Github, map[string]string{"GITHUB_REF": "refs/pull/123/merge"}, "123"},
		{"github non-pr ref ignored", "", vcsurl.Github, map[string]string{"GITHUB_REF": "refs/heads/main"}, ""},
		{"gitlab iid", "", vcsurl.Gitlab, map[string]string{"CI_MERGE_REQUEST_IID": "7"}, "7"},
		{"bitbucket id", "", vcsurl.Bitbucket, map[string]string{"BITBUCKET_PR_ID": "9"}, "9"},
		{"unknown tries all", "", vcsurl.UnknownVCS, map[string]string{"CI_MERGE_REQUEST_IID": "7"}, "7"},
		{"nothing set", "", vcsurl.Github, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_REF", "")
			t.Setenv("CI_MERGE_REQUEST_IID", "")
			t.Setenv("BITBUCKET_PR_ID", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := resolvePullRequestID(tt.flag, tt.vcs); got != tt.want {
				t.Errorf("resolvePullRequestID() = %q, want %q", got, tt.want)
			}
		})
	}
}
