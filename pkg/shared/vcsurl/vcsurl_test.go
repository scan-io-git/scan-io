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
			name:  "GitHub root of VCS URL",
			input: "https://github.com/",
			expected: VCSURL{
				Namespace:     "",
				Repository:    "",
				Branch:        "",
				HTTPRepoLink:  "",
				SSHRepoLink:   "",
				Raw:           "https://github.com/",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "Github HTTPS org URL",
			input: "https://github.com/scan-io-git/",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "",
				Branch:        "",
				HTTPRepoLink:  "",
				SSHRepoLink:   "",
				Raw:           "https://github.com/scan-io-git/",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "Github git@ repo URL",
			input: "git@github.com:scan-io-git/scan-io.git",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "scan-io",
				Branch:        "",
				HTTPRepoLink:  "https://github.com/scan-io-git/scan-io",
				SSHRepoLink:   "ssh://git@github.com/scan-io-git/scan-io.git",
				Raw:           "git@github.com:scan-io-git/scan-io.git",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "Github HTTPS .git repo URL",
			input: "https://github.com/scan-io-git/scan-io.git",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "scan-io",
				Branch:        "",
				HTTPRepoLink:  "https://github.com/scan-io-git/scan-io",
				SSHRepoLink:   "ssh://git@github.com/scan-io-git/scan-io.git",
				Raw:           "https://github.com/scan-io-git/scan-io.git",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "GitHub SSH repo URL",
			input: "ssh://git@github.com/scan-io-git/scan-io.git",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "scan-io",
				Branch:        "",
				HTTPRepoLink:  "https://github.com/scan-io-git/scan-io",
				SSHRepoLink:   "ssh://git@github.com/scan-io-git/scan-io.git",
				Raw:           "ssh://git@github.com/scan-io-git/scan-io.git",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "Github HTTPS repo URL",
			input: "https://github.com/scan-io-git/scan-io/",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "scan-io",
				Branch:        "",
				HTTPRepoLink:  "https://github.com/scan-io-git/scan-io",
				SSHRepoLink:   "ssh://git@github.com/scan-io-git/scan-io.git",
				Raw:           "https://github.com/scan-io-git/scan-io/",
				PullRequestId: "",
				VCSType:       Github,
			},
		},
		{
			name:  "Github HTTPS repo URL with Branch",
			input: "https://github.com/scan-io-git/scan-io/tree/scanio_bot/test/feature",
			expected: VCSURL{
				Namespace:     "scan-io-git",
				Repository:    "scan-io",
				Branch:        "scanio_bot/test/feature",
				HTTPRepoLink:  "https://github.com/scan-io-git/scan-io",
				SSHRepoLink:   "ssh://git@github.com/scan-io-git/scan-io.git",
				Raw:           "https://github.com/scan-io-git/scan-io/tree/scanio_bot/test/feature",
				PullRequestId: "",
				VCSType:       Github,
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

func TestParseGitLabURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected VCSURL
	}{
		{
			name:  "Gitlab root of VCS URL",
			input: "https://gitlab.com/",
			expected: VCSURL{
				Namespace:     "",
				Repository:    "",
				Branch:        "",
				HTTPRepoLink:  "",
				SSHRepoLink:   "",
				Raw:           "https://gitlab.com/",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
		{
			name:  "Gitlab HTTPS org URL",
			input: "https://gitlab.com/testing_scanio/",
			expected: VCSURL{
				Namespace:     "testing_scanio",
				Repository:    "",
				Branch:        "",
				HTTPRepoLink:  "",
				SSHRepoLink:   "",
				Raw:           "https://gitlab.com/testing_scanio/",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
		{
			name:  "Gitlab git@ repo URL",
			input: "git@gitlab.com:testing_scanio/testing_scanio.git",
			expected: VCSURL{
				Namespace:     "testing_scanio",
				Repository:    "testing_scanio",
				Branch:        "",
				HTTPRepoLink:  "https://gitlab.com/testing_scanio/testing_scanio",
				SSHRepoLink:   "ssh://git@gitlab.com/testing_scanio/testing_scanio.git",
				Raw:           "git@gitlab.com:testing_scanio/testing_scanio.git",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
		{
			name:  "Gitlab HTTPS .git repo URL",
			input: "https://gitlab.com/testing_scanio/testing_scanio.git",
			expected: VCSURL{
				Namespace:     "testing_scanio",
				Repository:    "testing_scanio",
				Branch:        "",
				HTTPRepoLink:  "https://gitlab.com/testing_scanio/testing_scanio",
				SSHRepoLink:   "ssh://git@gitlab.com/testing_scanio/testing_scanio.git",
				Raw:           "https://gitlab.com/testing_scanio/testing_scanio.git",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
		{
			name:  "Gitlab HTTPS repo URL",
			input: "https://gitlab.com/testing_scanio/testing_scanio",
			expected: VCSURL{
				Namespace:     "testing_scanio",
				Repository:    "testing_scanio",
				Branch:        "",
				HTTPRepoLink:  "https://gitlab.com/testing_scanio/testing_scanio",
				SSHRepoLink:   "ssh://git@gitlab.com/testing_scanio/testing_scanio.git",
				Raw:           "https://gitlab.com/testing_scanio/testing_scanio",
				PullRequestId: "",
				VCSType:       Gitlab,
			},
		},
		{
			name:  "Gitlab HTTPS repo URL with Branch",
			input: "https://gitlab.com/testing_scanio/testing_scanio/-/tree/test/feature",
			expected: VCSURL{
				Namespace:     "testing_scanio",
				Repository:    "testing_scanio",
				Branch:        "test/feature",
				HTTPRepoLink:  "https://gitlab.com/testing_scanio/testing_scanio",
				SSHRepoLink:   "ssh://git@gitlab.com/testing_scanio/testing_scanio.git",
				Raw:           "https://gitlab.com/testing_scanio/testing_scanio/-/tree/test/feature",
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
				Branch:        "",
				HTTPRepoLink:  "",
				SSHRepoLink:   "",
				Raw:           "https://bitbucket.org/",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket HTTPS APIv1 project URL",
			input: "https://bitbucket.org/projects/scanio-project",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "",
				Branch:        "",
				HTTPRepoLink:  "",
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
				Namespace:     "users/scanio-bot",
				Repository:    "scanio-test-repository",
				Branch:        "",
				HTTPRepoLink:  "https://bitbucket.org/users/scanio-bot/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:7989/~scanio-bot/scanio-test-repository.git",
				Raw:           "https://bitbucket.org/users/scanio-bot/repos/scanio-test-repository/",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket HTTPS APIv1 repo URL with Username and Branch",
			input: "https://bitbucket.org/users/scanio-bot/repos/scanio-test-repository/?at=refs%2Fheads%2Ftest%2Ffeature",
			expected: VCSURL{
				Namespace:     "users/scanio-bot",
				Repository:    "scanio-test-repository",
				Branch:        "refs/heads/test/feature",
				HTTPRepoLink:  "https://bitbucket.org/users/scanio-bot/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:7989/~scanio-bot/scanio-test-repository.git",
				Raw:           "https://bitbucket.org/users/scanio-bot/repos/scanio-test-repository/?at=refs%2Fheads%2Ftest%2Ffeature",
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
				Branch:        "",
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
				Branch:        "",
				HTTPRepoLink:  "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:7989/scanio-project/scanio-test-repository.git",
				Raw:           "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository/browse",
				PullRequestId: "",
				VCSType:       Bitbucket,
			},
		},
		{
			name:  "Bitbucket HTTPS APIv1 repo URL with Branch",
			input: "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository/browse?at=refs%2Fheads%2Ftest%2Ffeature",
			expected: VCSURL{
				Namespace:     "scanio-project",
				Repository:    "scanio-test-repository",
				Branch:        "refs/heads/test/feature",
				HTTPRepoLink:  "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository",
				SSHRepoLink:   "ssh://git@bitbucket.org:7989/scanio-project/scanio-test-repository.git",
				Raw:           "https://bitbucket.org/projects/scanio-project/repos/scanio-test-repository/browse?at=refs%2Fheads%2Ftest%2Ffeature",
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
				Branch:        "",
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
				Branch:        "",
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
				Branch:        "",
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
				Branch:        "",
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
				Branch:        "",
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
				Branch:        "",
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
				Branch:        "",
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
