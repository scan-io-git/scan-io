package tohtml

import (
	"os/exec"
	"strings"
	"testing"
)

const (
	provRepo   = "https://github.com/scan-io-git/scanio-demo"
	provBBRepo = "https://bitbucket.example.com/projects/KEY/repos/demo"
	provCommit = "0123456789abcdef0123456789abcdef01234567"
)

func provenanceResult() testResult {
	return testResult{Rule: "demo.sql-injection", URI: "app/db.py", Line: 12}
}

func TestProvenance_LinksWithoutGitCheckout(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", provenanceEntry(provRepo, provCommit, "feature/x"), provenanceResult())
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustContain(t, html, provRepo+"/blob/"+provCommit+"/app/db.py#L12")
	mustContain(t, html, "feature/x")
}

func TestProvenance_PRDiffLinkWithoutGitCheckout(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", provenanceEntry(provRepo, provCommit, "feature/x"), provenanceResult())
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir, PullRequest: "7"})

	mustContain(t, html, provRepo+"/pull/7/files#diff-")
}

// Bitbucket Server is the one provider besides GitHub the upstream producer of
// triage data runs on, so its permalink and PR-diff formats stay covered.
func TestProvenance_BitbucketServerRepositoryURI(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", provenanceEntry(provBBRepo, provCommit, "feature/x"), provenanceResult())
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir, PullRequest: "7"})

	mustContain(t, html, provBBRepo+"/browse/app/db.py?at="+provCommit)
	mustContain(t, html, provBBRepo+"/pull-requests/7/diff#app/db.py")
}

func TestProvenance_NoLinksWithoutAnySource(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", nil, provenanceResult())
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustNotContain(t, html, "/blob/")
	mustNotContain(t, html, "scan-io-git/scanio-demo")
}

// initGitRepo creates a repository in dir with one commit and, when origin is
// not empty, an origin remote. It returns the commit hash.
func initGitRepo(t *testing.T, dir, origin string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	steps := [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.email=a@example.com", "-c", "user.name=alice", "commit", "-q", "--allow-empty", "-m", "base"},
	}
	if origin != "" {
		steps = append(steps, []string{"remote", "add", "origin", origin})
	}
	for _, args := range steps {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestProvenance_GitWithoutOriginUsesProvenanceEntirely(t *testing.T) {
	dir := t.TempDir()
	localCommit := initGitRepo(t, dir, "")
	in := writeTestSARIF(t, dir, "in.sarif", provenanceEntry(provRepo, provCommit, "feature/x"), provenanceResult())
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustContain(t, html, provRepo+"/blob/"+provCommit+"/app/db.py#L12")
	mustContain(t, html, "feature/x")
	mustNotContain(t, html, localCommit)
}

func TestProvenance_GitWithoutOriginAndNoProvenanceRendersWithoutLinks(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir, "")
	for name, prov := range map[string][]map[string]any{
		"missing": nil,
		"empty":   {{"repositoryUri": "", "revisionId": provCommit}},
	} {
		t.Run(name, func(t *testing.T) {
			in := writeTestSARIF(t, dir, "in.sarif", prov, provenanceResult())
			html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})
			mustNotContain(t, html, "/browse/")
			mustNotContain(t, html, "/blob/")
		})
	}
}

func TestProvenance_UnparsableRepositoryURIRendersWithoutLinks(t *testing.T) {
	dir := t.TempDir()
	in := writeTestSARIF(t, dir, "in.sarif", provenanceEntry("not a url", provCommit, "main"), provenanceResult())
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustNotContain(t, html, "/browse/")
	mustNotContain(t, html, "/blob/")
	mustNotContain(t, html, "scan-io-git/scanio-demo")
	mustContain(t, html, `<details id="issue-0"`)
}

func TestProvenance_RejectedProvenanceKeepsLocalGitMetadata(t *testing.T) {
	dir := t.TempDir()
	localCommit := initGitRepo(t, dir, "")
	in := writeTestSARIF(t, dir, "in.sarif", provenanceEntry("not a url", provCommit, "feature/x"), provenanceResult())
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustContain(t, html, "<span>main</span>")
	mustContain(t, html, "<code>#"+localCommit[:7]+"</code>")
	mustContain(t, html, `data-report-state="`)
	mustNotContain(t, html, "feature/x")
	mustNotContain(t, html, provCommit[:7])
	mustNotContain(t, html, "/blob/")
}

func TestProvenance_GitMetadataWins(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir, "https://github.com/acme/real.git")
	in := writeTestSARIF(t, dir, "in.sarif", provenanceEntry(provRepo, provCommit, "feature/x"), provenanceResult())
	html := renderReport(t, ToHTMLOptions{Inputs: []string{in}, SourceFolder: dir})

	mustContain(t, html, "https://github.com/acme/real/blob/")
	mustNotContain(t, html, "scan-io-git/scanio-demo")
}
