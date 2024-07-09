package vcsurl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGitURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected VCSURL
	}{
		{
			name:  "GitHub git URL",
			input: "git@github.com:juice-shop/juice-shop.git",
			expected: VCSURL{
				Namespace:     "juice-shop",
				Repository:    "juice-shop",
				HTTPUrl:       "https://github.com/juice-shop/juice-shop",
				SSHUrl:        "ssh://git@github.com/juice-shop/juice-shop.git",
				Raw:           "git@github.com:juice-shop/juice-shop.git",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "GitLab git URL",
			input: "git@gitlab.com:scanio-demo/juice-shop.git",
			expected: VCSURL{
				Namespace:     "scanio-demo",
				Repository:    "juice-shop",
				HTTPUrl:       "https://gitlab.com/scanio-demo/juice-shop",
				SSHUrl:        "ssh://git@gitlab.com/scanio-demo/juice-shop.git",
				Raw:           "git@gitlab.com:scanio-demo/juice-shop.git",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
		// {
		// 	name:  "Bitbucket git URL",
		// 	input: "git@bitbucket.org:scanio/test-repository.git",
		// 	expected: VCSURL{
		// 		Namespace:     "scanio",
		// 		Repository:    "test-repository",
		// 		HTTPUrl:       "https://bitbucket.org/scanio/test-repository",
		// 		SSHUrl:        "ssh://git@bitbucket.org/scanio/test-repository.git",
		// 		Raw:           "git@bitbucket.org:scanio/test-repository.git",
		// 		PullRequestId: "",
		// 		VCSType:       Bitbucket,
		// 	},
		// },
		{
			name:  "Github HTTP URL",
			input: "https://github.com/juice-shop/juice-shop.git",
			expected: VCSURL{
				Namespace:     "juice-shop",
				Repository:    "juice-shop",
				HTTPUrl:       "https://github.com/juice-shop/juice-shop",
				SSHUrl:        "ssh://git@github.com/juice-shop/juice-shop.git",
				Raw:           "https://github.com/juice-shop/juice-shop.git",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "Github HTTP URL",
			input: "https://github.com/juice-shop/juice-shop.git",
			expected: VCSURL{
				Namespace:     "juice-shop",
				Repository:    "juice-shop",
				HTTPUrl:       "https://github.com/juice-shop/juice-shop",
				SSHUrl:        "ssh://git@github.com/juice-shop/juice-shop.git",
				Raw:           "https://github.com/juice-shop/juice-shop.git",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		// {
		// 	name:  "Bitbucket Custom URL",
		// 	input: "http://bitbucket.example.com:7994/scm/thet/projectsource.git",
		// 	expected: VCSURL{
		// 		Namespace:     "thet",
		// 		Repository:    "projectsource",
		// 		HTTPUrl:       "http://bitbucket.example.com:7994/scm/thet/projectsource",
		// 		SSHUrl:        "ssh://git@bitbucket.example.com:7994/thet/projectsource.git",
		// 		Raw:           "http://bitbucket.example.com:7994/scm/thet/projectsource.git",
		// 		PullRequestId: "",
		// 		VCSType:       Bitbucket,
		// 	},
		// },
		{
			name:  "GitLab HTTPS URL",
			input: "https://gitlab.com/juice-shop/juice-shop.git",
			expected: VCSURL{
				Namespace:     "juice-shop",
				Repository:    "juice-shop",
				HTTPUrl:       "https://gitlab.com/juice-shop/juice-shop",
				SSHUrl:        "ssh://git@gitlab.com/juice-shop/juice-shop.git",
				Raw:           "https://gitlab.com/juice-shop/juice-shop.git",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
		// {
		// 	name:  "Bitbucket HTTPS URL with Username",
		// 	input: "https://japroc@bitbucket.org/scanio/test-repository.git",
		// 	expected: VCSURL{
		// 		Namespace:     "scanio",
		// 		Repository:    "test-repository",
		// 		HTTPUrl:       "https://bitbucket.org/scanio/test-repository",
		// 		SSHUrl:        "ssh://git@bitbucket.org/scanio/test-repository.git",
		// 		Raw:           "https://japroc@bitbucket.org/scanio/test-repository.git",
		// 		PullRequestId: "",
		// 		VCSType:       Bitbucket,
		// 	},
		// },
		{
			name:  "GitHub SSH URL",
			input: "ssh://git@github.com/scan-io-git/scan-io.git",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "scan-io",
				HTTPUrl:       "https://github.com/scan-io-git/scan-io",
				SSHUrl:        "ssh://git@github.com/scan-io-git/scan-io.git",
				Raw:           "ssh://git@github.com/scan-io-git/scan-io.git",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)
			assert.NoError(t, err, "Parse should not return an error")

			assert.Equal(t, tc.expected.Namespace, got.Namespace, "Namespace mismatch")
			assert.Equal(t, tc.expected.Repository, got.Repository, "Repository mismatch")
			assert.Equal(t, tc.expected.HTTPUrl, got.HTTPUrl, "HTTPUrl mismatch")
			assert.Equal(t, tc.expected.SSHUrl, got.SSHUrl, "SSHUrl mismatch")
			assert.Equal(t, tc.expected.Raw, got.Raw, "Raw input mismatch")
			assert.Equal(t, tc.expected.PullRequestId, got.PullRequestId, "PullRequestId mismatch")
			assert.Equal(t, tc.expected.VCSType, got.VCSType, "VCSType mismatch")
			assert.NotNil(t, got.ParsedURL, "ParsedURL should not be nil")
		})
	}
}
