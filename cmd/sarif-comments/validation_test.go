package sarifcomments

import (
	"testing"

	cmdutil "github.com/scan-io-git/scan-io/internal/cmd"
	"github.com/scan-io-git/scan-io/internal/vcsintegrator"
)

func TestValidateSarifCommentsArgs_FlagsMode(t *testing.T) {
	t.Run("unexpected positional arguments", func(t *testing.T) {
		opts := &vcsintegrator.RunOptionsIntegrationVCS{}
		err := validateSarifCommentsArgs(opts, []string{"extra"}, cmdutil.ModeFlags)
		if err == nil || err.Error() != "unexpected positional arguments: extra; missing required flags: vcs, domain, namespace, repository, pull-request-id, sarif, source" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing required flags", func(t *testing.T) {
		opts := &vcsintegrator.RunOptionsIntegrationVCS{}
		err := validateSarifCommentsArgs(opts, nil, cmdutil.ModeFlags)
		if err == nil {
			t.Fatal("expected error for missing flags")
		}
		want := "missing required flags: vcs, domain, namespace, repository, pull-request-id, sarif, source"
		if err.Error() != want {
			t.Fatalf("unexpected error message\nwant: %q\n got: %q", want, err.Error())
		}
	})

	t.Run("limit cannot be negative", func(t *testing.T) {
		opts := &vcsintegrator.RunOptionsIntegrationVCS{
			VCSPluginName:    "bitbucket",
			Domain:           "example.com",
			Namespace:        "team",
			Repository:       "repo",
			PullRequestID:    "123",
			SarifInput:       "report.sarif",
			SourceFolder:     "src",
			SarifIssuesLimit: -1,
		}
		err := validateSarifCommentsArgs(opts, nil, cmdutil.ModeFlags)
		if err == nil {
			t.Fatal("expected error for negative limit")
		}
		if err.Error() != "'limit' cannot be negative" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid options", func(t *testing.T) {
		opts := &vcsintegrator.RunOptionsIntegrationVCS{
			VCSPluginName:    "bitbucket",
			Domain:           "example.com",
			Namespace:        "team",
			Repository:       "repo",
			PullRequestID:    "123",
			SarifInput:       "report.sarif",
			SourceFolder:     "src",
			SarifIssuesLimit: 10,
		}
		if err := validateSarifCommentsArgs(opts, nil, cmdutil.ModeFlags); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("aggregates missing flags and invalid limit", func(t *testing.T) {
		opts := &vcsintegrator.RunOptionsIntegrationVCS{
			VCSPluginName:    "bitbucket",
			SarifInput:       "report.sarif",
			SarifIssuesLimit: -5,
		}
		err := validateSarifCommentsArgs(opts, nil, cmdutil.ModeFlags)
		if err == nil {
			t.Fatal("expected aggregated error")
		}
		want := "missing required flags: domain, namespace, repository, pull-request-id, source; 'limit' cannot be negative"
		if err.Error() != want {
			t.Fatalf("unexpected aggregated error\nwant: %q\n got: %q", want, err.Error())
		}
	})
}

func TestValidateSarifCommentsArgs_SingleURLMode(t *testing.T) {
	t.Run("requires single URL", func(t *testing.T) {
		opts := &vcsintegrator.RunOptionsIntegrationVCS{}
		err := validateSarifCommentsArgs(opts, []string{"url1", "url2"}, cmdutil.ModeSingleURL)
		if err == nil || err.Error() != "provide exactly one repository URL; missing required flags: vcs, sarif, source" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("requires plugin and sarif", func(t *testing.T) {
		opts := &vcsintegrator.RunOptionsIntegrationVCS{}
		err := validateSarifCommentsArgs(opts, []string{"https://example.com"}, cmdutil.ModeSingleURL)
		if err == nil {
			t.Fatal("expected error")
		}
		want := "missing required flags: vcs, sarif, source"
		if err.Error() != want {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid single URL", func(t *testing.T) {
		opts := &vcsintegrator.RunOptionsIntegrationVCS{
			VCSPluginName:    "bitbucket",
			SarifInput:       "report.sarif",
			SourceFolder:     "src",
			SarifIssuesLimit: 0,
		}
		if err := validateSarifCommentsArgs(opts, []string{"https://example.com"}, cmdutil.ModeSingleURL); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
