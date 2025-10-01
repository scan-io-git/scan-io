package ci

import (
	"errors"
	"testing"
)

func TestCIKindString(t *testing.T) {
	testCases := []struct {
		name string
		kind CIKind
		want string
	}{
		{name: "GitHub", kind: CIGitHub, want: "github"},
		{name: "GitLab", kind: CIGitLab, want: "gitlab"},
		{name: "Bitbucket", kind: CIBitbucket, want: "bitbucket"},
		{name: "Unknown", kind: CIUnknown, want: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.kind.String(); got != tc.want {
				t.Fatalf("CIKind.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseCIKind(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    CIKind
		wantErr error
	}{
		{name: "GitHub", input: "github", want: CIGitHub},
		{name: "GitLab", input: " GitLab ", want: CIGitLab},
		{name: "Bitbucket", input: "BITBUCKET", want: CIBitbucket},
		{name: "Unsupported", input: "ado", want: CIUnknown, wantErr: errors.New("unsupported")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCIKind(tc.input)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("ParseCIKind(%q) expected error", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseCIKind(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("ParseCIKind(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestGetCIDefaultEnvVars(t *testing.T) {
	t.Run("GitHub", func(t *testing.T) {
		env := map[string]string{
			"CI":                      "true",
			"GITHUB_REPOSITORY":       "octocat/hello-world",
			"GITHUB_SERVER_URL":       "https://github.example.com",
			"GITHUB_SHA":              "abcdef123456",
			"GITHUB_REF":              "refs/heads/main",
			"GITHUB_REF_NAME":         "main",
			"GITHUB_REPOSITORY_OWNER": "octocat",
		}

		lookup := mapLookup(env)
		got, err := getCIDefaultEnvVars(CIGitHub, lookup)
		if err != nil {
			t.Fatalf("getCIDefaultEnvVars() error = %v", err)
		}

		want := CIEnvironment{
			Kind:               CIGitHub,
			CI:                 true,
			CommitHash:         "abcdef123456",
			VCSServerURL:       "https://github.example.com",
			Reference:          "refs/heads/main",
			ReferenceName:      "main",
			RepositoryName:     "hello-world",
			RepositoryFullName: "octocat/hello-world",
			RepositoryFullPath: "https://github.example.com/octocat/hello-world",
			Namespace:          "octocat",
		}

		if got != want {
			t.Fatalf("GitHub env = %+v, want %+v", got, want)
		}
	})

	t.Run("GitLabMergeRequest", func(t *testing.T) {
		env := map[string]string{
			"CI":                        "true",
			"CI_COMMIT_SHA":             "deadbeef",
			"CI_SERVER_URL":             "https://gitlab.example.com",
			"CI_MERGE_REQUEST_REF_PATH": "refs/merge-requests/42/head",
			"CI_MERGE_REQUEST_IID":      "42",
			"CI_PROJECT_NAME":           "demo",
			"CI_PROJECT_PATH":           "group/demo",
			"CI_PROJECT_URL":            "https://gitlab.example.com/group/demo",
			"CI_PROJECT_NAMESPACE":      "group",
		}

		lookup := mapLookup(env)
		got, err := getCIDefaultEnvVars(CIGitLab, lookup)
		if err != nil {
			t.Fatalf("getCIDefaultEnvVars() error = %v", err)
		}

		want := CIEnvironment{
			Kind:               CIGitLab,
			CI:                 true,
			CommitHash:         "deadbeef",
			VCSServerURL:       "https://gitlab.example.com",
			Reference:          "refs/merge-requests/42/head",
			ReferenceName:      "42",
			RepositoryName:     "demo",
			RepositoryFullName: "group/demo",
			RepositoryFullPath: "https://gitlab.example.com/group/demo",
			Namespace:          "group",
		}

		if got != want {
			t.Fatalf("GitLab env = %+v, want %+v", got, want)
		}
	})

	t.Run("BitbucketPullRequest", func(t *testing.T) {
		env := map[string]string{
			"CI":                        "true",
			"BITBUCKET_COMMIT":          "1234567",
			"BITBUCKET_GIT_HTTP_ORIGIN": "https://bitbucket.org/workspace/repo",
			"BITBUCKET_PR_ID":           "7",
			"BITBUCKET_REPO_SLUG":       "repo",
			"BITBUCKET_REPO_FULL_NAME":  "workspace/repo",
			"BITBUCKET_WORKSPACE":       "workspace",
		}

		lookup := mapLookup(env)
		got, err := getCIDefaultEnvVars(CIBitbucket, lookup)
		if err != nil {
			t.Fatalf("getCIDefaultEnvVars() error = %v", err)
		}

		want := CIEnvironment{
			Kind:               CIBitbucket,
			CI:                 true,
			CommitHash:         "1234567",
			VCSServerURL:       "https://bitbucket.org",
			Reference:          "refs/pull/7",
			ReferenceName:      "7",
			RepositoryName:     "repo",
			RepositoryFullName: "workspace/repo",
			RepositoryFullPath: "https://bitbucket.org/workspace/repo",
			Namespace:          "workspace",
		}

		if got != want {
			t.Fatalf("Bitbucket env = %+v, want %+v", got, want)
		}
	})

	t.Run("UnknownKind", func(t *testing.T) {
		if _, err := getCIDefaultEnvVars(CIUnknown, mapLookup(nil)); err == nil {
			t.Fatalf("expected error when kind is CIUnknown")
		}
	})
}

func mapLookup(values map[string]string) LookupFunc {
	return func(key string) string {
		if values == nil {
			return ""
		}
		return values[key]
	}
}
