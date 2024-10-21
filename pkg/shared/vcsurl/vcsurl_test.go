package vcsurl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validateParse(t *testing.T, expected *VCSURL, got *VCSURL) {
	assert.Equal(t, expected.Namespace, got.Namespace, "Namespace mismatch")
	assert.Equal(t, expected.Repository, got.Repository, "Repository mismatch")
	assert.Equal(t, expected.HTTPRepoLink, got.HTTPRepoLink, "HTTPRepoLink mismatch")
	assert.Equal(t, expected.SSHRepoLink, got.SSHRepoLink, "SSHRepoLink mismatch")
	assert.Equal(t, expected.Raw, got.Raw, "Raw input mismatch")
	assert.Equal(t, expected.PullRequestId, got.PullRequestId, "PullRequestId mismatch")
	assert.Equal(t, expected.VCSType, got.VCSType, "VCSType mismatch")
	assert.NotNil(t, got.ParsedURL, "ParsedURL should not be nil")
}

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
				HTTPRepoLink:  "https://github.com/juice-shop/juice-shop",
				SSHRepoLink:   "ssh://git@github.com/juice-shop/juice-shop.git",
				Raw:           "git@github.com:juice-shop/juice-shop.git",
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
				HTTPRepoLink:  "https://github.com/juice-shop/juice-shop",
				SSHRepoLink:   "ssh://git@github.com/juice-shop/juice-shop.git",
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
				HTTPRepoLink:  "https://github.com/juice-shop/juice-shop",
				SSHRepoLink:   "ssh://git@github.com/juice-shop/juice-shop.git",
				Raw:           "https://github.com/juice-shop/juice-shop.git",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "GitHub SSH URL",
			input: "ssh://git@github.com/scan-io-git/scan-io.git",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "scan-io",
				HTTPRepoLink:  "https://github.com/scan-io-git/scan-io",
				SSHRepoLink:   "ssh://git@github.com/scan-io-git/scan-io.git",
				Raw:           "ssh://git@github.com/scan-io-git/scan-io.git",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "Github HTTP URL 2",
			input: "https://github.com/scan-io-git/scan-io/",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "scan-io",
				HTTPRepoLink:  "https://github.com/scan-io-git/scan-io",
				SSHRepoLink:   "ssh://git@github.com/scan-io-git/scan-io.git",
				Raw:           "https://github.com/scan-io-git/scan-io/",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "Github SSH URL 2",
			input: "git@github.com:scan-io-git/scan-io.git",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "scan-io",
				HTTPRepoLink:  "https://github.com/scan-io-git/scan-io",
				SSHRepoLink:   "ssh://git@github.com/scan-io-git/scan-io.git",
				Raw:           "git@github.com:scan-io-git/scan-io.git",
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
				HTTPRepoLink:  "https://gitlab.com/scanio-demo/juice-shop",
				SSHRepoLink:   "ssh://git@gitlab.com/scanio-demo/juice-shop.git",
				Raw:           "git@gitlab.com:scanio-demo/juice-shop.git",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
		{
			name:  "GitLab HTTPS URL",
			input: "https://gitlab.com/juice-shop/juice-shop.git",
			expected: VCSURL{
				Namespace:     "juice-shop",
				Repository:    "juice-shop",
				HTTPRepoLink:  "https://gitlab.com/juice-shop/juice-shop",
				SSHRepoLink:   "ssh://git@gitlab.com/juice-shop/juice-shop.git",
				Raw:           "https://gitlab.com/juice-shop/juice-shop.git",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
		{
			name:  "GitLab web URL",
			input: "https://gitlab.com/scanio-demo/juice-shop",
			expected: VCSURL{
				Namespace:     "scanio-demo",
				Repository:    "juice-shop",
				HTTPRepoLink:  "https://gitlab.com/scanio-demo/juice-shop",
				SSHRepoLink:   "ssh://git@gitlab.com/scanio-demo/juice-shop.git",
				Raw:           "https://gitlab.com/scanio-demo/juice-shop",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)
			assert.NoError(t, err, "Parse should not return an error")

			validateParse(t, &tc.expected, got)
		})
	}
}

func TestParseBitbucketAPIV1(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected VCSURL
	}{
		{
			name:  "Bitbucket HTTPS APIv1 root of VCS URL",
			input: "https://bitbucket.org/",
			expected: VCSURL{
				Namespace:     "",
				Repository:    "",
				HTTPRepoLink:  "",
				SSHRepoLink:   "",
				Raw:           "https://bitbucket.org/",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket HTTPS APIv1 project URL", // fixed
			input: "https://bitbucket.org/projects/scanio-project",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "",
				HTTPRepoLink:  "", // delete u.raw
				SSHRepoLink:   "",
				Raw:           "https://bitbucket.org/projects/scanio-project",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket HTTPS APIv1 repo URL with Username",
			input: "https://bitbucket.org/users/scanio-bot/repos/scanio-test-repository/",
			expected: VCSURL{
				Namespace:     "scanio-bot",
				Repository:    "scanio-test-repository",
				HTTPRepoLink:  "https://bitbucket.org/users/scanio-bot/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:7989/~scanio-bot/scanio-test-repository.git",
				Raw:           "https://bitbucket.org/users/scanio-bot/repos/scanio-test-repository/",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket HTTPS APIv1 pullrequest URL",
			input: "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository/pull-requests/1",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "scanio-test-repository",
				HTTPRepoLink:  "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:7989/scanio-project/scanio-test-repository.git",
				Raw:           "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository/pull-requests/1",
				PullRequestId: "1",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket HTTPS APIv1 repo URL",
			input: "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository/browse",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "scanio-test-repository",
				HTTPRepoLink:  "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:7989/scanio-project/scanio-test-repository.git",
				Raw:           "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository/browse",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},

		{
			name:  "Bitbucket HTTPS APIv1 scm project URL",
			input: "https://bitbucket.org/scm/scanio-project",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "",
				HTTPRepoLink:  "",
				SSHRepoLink:   "",
				Raw:           "https://bitbucket.org/scm/scanio-project",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket HTTPS APIv1 scm repo URL",
			input: "https://bitbucket.org/scm/scanio-project/scanio-test-repository.git",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "scanio-test-repository",
				HTTPRepoLink:  "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:7989/scanio-project/scanio-test-repository.git",
				Raw:           "https://bitbucket.org/scm/scanio-project/scanio-test-repository.git",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket APIv1 SSH project URL",
			input: "ssh://git@bitbucket.org/scanio-project/",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "",
				HTTPRepoLink:  "",
				SSHRepoLink:   "",
				Raw:           "ssh://git@bitbucket.org/scanio-project/",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket APIv1 SSH repo no port URL",
			input: "ssh://git@bitbucket.org/scanio-project/scanio-test-repository.git",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "scanio-test-repository",
				HTTPRepoLink:  "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:7989/scanio-project/scanio-test-repository.git",
				Raw:           "ssh://git@bitbucket.org/scanio-project/scanio-test-repository.git",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket APIv1 SSH repo with port URL",
			input: "ssh://git@bitbucket.org:22/scanio-project/scanio-test-repository.git",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "scanio-test-repository",
				HTTPRepoLink:  "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:22/scanio-project/scanio-test-repository.git",
				Raw:           "ssh://git@bitbucket.org:22/scanio-project/scanio-test-repository.git",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)
			assert.NoError(t, err, "Parse should not return an error")

			validateParse(t, &tc.expected, got)
		})
	}
}

func TestGenericVCSForBitbucket(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected VCSURL
	}{
		{
			name:  "Bitbucket git URL",
			input: "git@bitbucket.org:scanio/test-repository.git",
			expected: VCSURL{
				Namespace:     "scanio",
				Repository:    "test-repository",
				HTTPRepoLink:  "https://bitbucket.org/scanio/test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org/scanio/test-repository.git",
				Raw:           "git@bitbucket.org:scanio/test-repository.git",
				PullRequestId: "",
				VCSType:       GenericVCS,
			},
		},
		{
			name:  "Bitbucket HTTPS URL with Username",
			input: "https://japroc@bitbucket.org/scanio/test-repository.git",
			expected: VCSURL{
				Namespace:     "scanio",
				Repository:    "test-repository",
				HTTPRepoLink:  "https://bitbucket.org/scanio/test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org/scanio/test-repository.git",
				Raw:           "https://japroc@bitbucket.org/scanio/test-repository.git",
				PullRequestId: "",
				VCSType:       GenericVCS,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseForVCSType(tc.input, GenericVCS)
			assert.NoError(t, err, "Parse should not return an error")

			validateParse(t, &tc.expected, got)
		})
	}
}
