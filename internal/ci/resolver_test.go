package ci

import "testing"

func clearResolverEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"GITHUB_REPOSITORY",
		"GITHUB_SERVER_URL",
		"GITHUB_SHA",
		"GITHUB_REF",
		"GITHUB_REF_NAME",
		"GITHUB_REPOSITORY_OWNER",
		"CI",
		"GITLAB_CI",
		"CI_PROJECT_PATH",
		"CI_PROJECT_NAME",
		"CI_PROJECT_NAMESPACE",
		"CI_PROJECT_URL",
		"CI_SERVER_URL",
		"CI_COMMIT_SHA",
		"CI_COMMIT_REF_NAME",
		"CI_MERGE_REQUEST_REF_PATH",
		"CI_MERGE_REQUEST_IID",
		"BITBUCKET_WORKSPACE",
		"BITBUCKET_REPO_SLUG",
		"BITBUCKET_REPO_FULL_NAME",
		"BITBUCKET_GIT_HTTP_ORIGIN",
		"BITBUCKET_COMMIT",
		"BITBUCKET_PR_ID",
		"BITBUCKET_BRANCH",
		"BITBUCKET_TAG",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func TestResolveFromEnvironment_GitHubDetection(t *testing.T) {
	clearResolverEnv(t)

	t.Setenv("GITHUB_REPOSITORY", "octocat/hello-world")
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_SHA", "abcdef")
	t.Setenv("GITHUB_REF", "refs/pull/42/merge")
	t.Setenv("GITHUB_REF_NAME", "42")
	t.Setenv("GITHUB_REPOSITORY_OWNER", "octocat")
	t.Setenv("CI", "true")

	res, err := ResolveFromEnvironment(nil, "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.PluginName != "github" {
		t.Fatalf("expected plugin github, got %q", res.PluginName)
	}
	if res.Kind != CIGitHub {
		t.Fatalf("expected kind github, got %v", res.Kind)
	}
	if res.Domain != "github.com" {
		t.Fatalf("expected domain github.com, got %q", res.Domain)
	}
	if res.Namespace != "octocat" {
		t.Fatalf("expected namespace octocat, got %q", res.Namespace)
	}
	if res.Repository != "hello-world" {
		t.Fatalf("expected repository hello-world, got %q", res.Repository)
	}
	if res.PullRequest != "42" {
		t.Fatalf("expected pull request 42, got %q", res.PullRequest)
	}
	if !res.Hydrated {
		t.Fatalf("expected hydrated to be true")
	}
}

func TestResolveFromEnvironment_GitLabProvided(t *testing.T) {
	clearResolverEnv(t)

	t.Setenv("GITLAB_CI", "true")
	t.Setenv("CI_PROJECT_PATH", "group/project")
	t.Setenv("CI_PROJECT_NAME", "project")
	t.Setenv("CI_PROJECT_NAMESPACE", "group")
	t.Setenv("CI_PROJECT_URL", "https://gitlab.example.com/group/project")
	t.Setenv("CI_SERVER_URL", "https://gitlab.example.com")
	t.Setenv("CI_COMMIT_SHA", "deadbeef")
	t.Setenv("CI_MERGE_REQUEST_REF_PATH", "refs/merge-requests/7/head")
	t.Setenv("CI_MERGE_REQUEST_IID", "7")

	res, err := ResolveFromEnvironment(nil, "gitlab")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.PluginName != "gitlab" {
		t.Fatalf("expected plugin gitlab, got %q", res.PluginName)
	}
	if res.Domain != "gitlab.example.com" {
		t.Fatalf("expected domain gitlab.example.com, got %q", res.Domain)
	}
	if res.Namespace != "group" {
		t.Fatalf("expected namespace group, got %q", res.Namespace)
	}
	if res.Repository != "project" {
		t.Fatalf("expected repository project, got %q", res.Repository)
	}
	if res.PullRequest != "7" {
		t.Fatalf("expected pull request 7, got %q", res.PullRequest)
	}
}

func TestResolveFromEnvironment_UnsupportedProvided(t *testing.T) {
	clearResolverEnv(t)

	res, err := ResolveFromEnvironment(nil, "ado")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.PluginName != "ado" {
		t.Fatalf("expected plugin to remain ado, got %q", res.PluginName)
	}
	if res.Hydrated {
		t.Fatalf("expected hydrated to be false")
	}
}

func TestResolveFromEnvironment_ErrorWhenUnknownAndMissing(t *testing.T) {
	clearResolverEnv(t)

	if _, err := ResolveFromEnvironment(nil, ""); err == nil {
		t.Fatalf("expected error when plugin not provided and CI is unknown")
	}
}
