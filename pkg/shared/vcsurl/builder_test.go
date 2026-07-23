package vcsurl

import (
	"net/url"
	"testing"
)

// bbDC builds a Bitbucket DC VCSURL for tests.
func bbDC(httpRepoLink, prID string) *VCSURL {
	return &VCSURL{
		VCSType:       Bitbucket,
		Namespace:     "SEC",
		Repository:    "sast-vulnerable-service",
		HTTPRepoLink:  httpRepoLink,
		PullRequestId: prID,
	}
}

// ghURL builds a GitHub VCSURL for tests.
func ghURL(httpRepoLink, prID string) *VCSURL {
	return &VCSURL{
		VCSType:       Github,
		Namespace:     "scan-io-git",
		Repository:    "scan-io",
		HTTPRepoLink:  httpRepoLink,
		PullRequestId: prID,
	}
}

// glURL builds a GitLab VCSURL for tests.
func glURL(httpRepoLink, prID string) *VCSURL {
	return &VCSURL{
		VCSType:       Gitlab,
		Namespace:     "sec",
		Repository:    "scan-io",
		HTTPRepoLink:  httpRepoLink,
		PullRequestId: prID,
	}
}

func TestFilePermalink(t *testing.T) {
	const sha = "abc1234def5678"
	base := "https://git.example-corp.internal/projects/SEC/repos/sast-vulnerable-service"

	tests := []struct {
		name      string
		u         *VCSURL
		ref       string
		filePath  string
		startLine int
		endLine   int
		want      string
	}{
		{
			name:      "bitbucket multi-line range",
			u:         bbDC(base, ""),
			ref:       sha,
			filePath:  "nginx-misconfigurations/vuln.conf",
			startLine: 15,
			endLine:   20,
			want:      base + "/browse/nginx-misconfigurations/vuln.conf?at=" + sha + "#15-20",
		},
		{
			name:      "bitbucket single line",
			u:         bbDC(base, ""),
			ref:       sha,
			filePath:  "src/main.go",
			startLine: 42,
			endLine:   42,
			want:      base + "/browse/src/main.go?at=" + sha + "#42",
		},
		{
			name:      "bitbucket no line",
			u:         bbDC(base, ""),
			ref:       sha,
			filePath:  "README.md",
			startLine: 0,
			endLine:   0,
			want:      base + "/browse/README.md?at=" + sha,
		},
		{
			name:      "bitbucket end less than start",
			u:         bbDC(base, ""),
			ref:       sha,
			filePath:  "src/main.go",
			startLine: 10,
			endLine:   5,
			want:      base + "/browse/src/main.go?at=" + sha + "#10",
		},
		{
			name:      "github multi-line range",
			u:         ghURL("https://github.com/scan-io-git/scan-io", ""),
			ref:       sha,
			filePath:  "internal/sarif/path_helpers.go",
			startLine: 10,
			endLine:   20,
			want:      "https://github.com/scan-io-git/scan-io/blob/" + sha + "/internal/sarif/path_helpers.go#L10-L20",
		},
		{
			name:      "github single line",
			u:         ghURL("https://github.com/scan-io-git/scan-io", ""),
			ref:       "main",
			filePath:  "src/main.go",
			startLine: 42,
			endLine:   0,
			want:      "https://github.com/scan-io-git/scan-io/blob/main/src/main.go#L42",
		},
		{
			name:      "github no line",
			u:         ghURL("https://github.com/scan-io-git/scan-io", ""),
			ref:       "main",
			filePath:  "README.md",
			startLine: 0,
			endLine:   0,
			want:      "https://github.com/scan-io-git/scan-io/blob/main/README.md",
		},
		{
			name:      "gitlab multi-line range",
			u:         glURL("https://gitlab.com/sec/scan-io", ""),
			ref:       sha,
			filePath:  "internal/sarif/sarif.go",
			startLine: 10,
			endLine:   20,
			want:      "https://gitlab.com/sec/scan-io/-/blob/" + sha + "/internal/sarif/sarif.go#L10-20",
		},
		{
			name:      "gitlab single line",
			u:         glURL("https://gitlab.com/sec/scan-io", ""),
			ref:       "main",
			filePath:  "main.go",
			startLine: 5,
			endLine:   5,
			want:      "https://gitlab.com/sec/scan-io/-/blob/main/main.go#L5",
		},
		{
			name:     "empty ref returns empty",
			u:        ghURL("https://github.com/scan-io-git/scan-io", ""),
			ref:      "",
			filePath: "main.go",
			want:     "",
		},
		{
			name:      "empty file returns empty",
			u:         ghURL("https://github.com/scan-io-git/scan-io", ""),
			ref:       "main",
			filePath:  "",
			startLine: 1,
			want:      "",
		},
		{
			name:      "backslash normalized",
			u:         ghURL("https://github.com/scan-io-git/scan-io", ""),
			ref:       "main",
			filePath:  `src\main.go`,
			startLine: 1,
			endLine:   1,
			want:      "https://github.com/scan-io-git/scan-io/blob/main/src/main.go#L1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.u.FilePermalink(tt.ref, tt.filePath, tt.startLine, tt.endLine)
			if got != tt.want {
				t.Errorf("FilePermalink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPRDiffLink(t *testing.T) {
	base := "https://git.example-corp.internal/projects/SEC/repos/sast-vulnerable-service"

	tests := []struct {
		name     string
		u        *VCSURL
		filePath string
		line     int
		want     string
	}{
		{
			name:     "bitbucket pr diff single line",
			u:        bbDC(base, "5"),
			filePath: "nginx-misconfigurations/example.com/html/csp/csp.html",
			line:     15,
			want:     base + "/pull-requests/5/diff#nginx-misconfigurations/example.com/html/csp/csp.html?t=15",
		},
		{
			name:     "bitbucket no pr id returns empty",
			u:        bbDC(base, ""),
			filePath: "src/main.go",
			line:     5,
			want:     "",
		},
		{
			name:     "bitbucket line zero returns empty",
			u:        bbDC(base, "5"),
			filePath: "src/main.go",
			line:     0,
			want:     "",
		},
		{
			name:     "github pr diff line-precise",
			u:        ghURL("https://github.com/scan-io-git/scan-io", "42"),
			filePath: "src/main.go",
			line:     10,
			want:     "https://github.com/scan-io-git/scan-io/pull/42/files#diff-9e185f29fa355d7dd8fdd9c9ff1d0723b85206aa7d37c4eec93997005dc291ebR10",
		},
		{
			name:     "github line zero falls back to files tab",
			u:        ghURL("https://github.com/scan-io-git/scan-io", "42"),
			filePath: "src/main.go",
			line:     0,
			want:     "https://github.com/scan-io-git/scan-io/pull/42/files",
		},
		{
			name:     "github no pr id returns empty",
			u:        ghURL("https://github.com/scan-io-git/scan-io", ""),
			filePath: "src/main.go",
			line:     10,
			want:     "",
		},
		{
			name:     "gitlab mr diff line-precise",
			u:        glURL("https://gitlab.com/sec/scan-io", "7"),
			filePath: "main.go",
			line:     3,
			want:     "https://gitlab.com/sec/scan-io/-/merge_requests/7/diffs#0607f785dfa3c3861b3239f6723eb276d8056461_3_3",
		},
		{
			name:     "gitlab line zero falls back to diffs tab",
			u:        glURL("https://gitlab.com/sec/scan-io", "7"),
			filePath: "main.go",
			line:     0,
			want:     "https://gitlab.com/sec/scan-io/-/merge_requests/7/diffs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.u.PRDiffLink(tt.filePath, tt.line)
			if got != tt.want {
				t.Errorf("PRDiffLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBranchURL(t *testing.T) {
	tests := []struct {
		name   string
		u      *VCSURL
		branch string
		want   string
	}{
		{
			name:   "bitbucket branch url",
			u:      bbDC("https://git.example-corp.internal/projects/SEC/repos/repo", ""),
			branch: "main",
			want:   "https://git.example-corp.internal/projects/SEC/repos/repo/browse?at=refs%2Fheads%2Fmain",
		},
		{
			name:   "github branch url",
			u:      ghURL("https://github.com/scan-io-git/scan-io", ""),
			branch: "feature/x",
			want:   "https://github.com/scan-io-git/scan-io/tree/feature/x",
		},
		{
			name:   "gitlab branch url",
			u:      glURL("https://gitlab.com/sec/scan-io", ""),
			branch: "develop",
			want:   "https://gitlab.com/sec/scan-io/-/tree/develop",
		},
		{
			name:   "empty branch returns empty",
			u:      ghURL("https://github.com/scan-io-git/scan-io", ""),
			branch: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.u.BranchURL(tt.branch)
			if got != tt.want {
				t.Errorf("BranchURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCommitURL(t *testing.T) {
	const sha = "abc1234"

	tests := []struct {
		name string
		u    *VCSURL
		sha  string
		want string
	}{
		{
			name: "bitbucket commit url",
			u:    bbDC("https://git.example-corp.internal/projects/SEC/repos/repo", ""),
			sha:  sha,
			want: "https://git.example-corp.internal/projects/SEC/repos/repo/commits/" + sha,
		},
		{
			name: "github commit url",
			u:    ghURL("https://github.com/scan-io-git/scan-io", ""),
			sha:  sha,
			want: "https://github.com/scan-io-git/scan-io/commit/" + sha,
		},
		{
			name: "gitlab commit url",
			u:    glURL("https://gitlab.com/sec/scan-io", ""),
			sha:  sha,
			want: "https://gitlab.com/sec/scan-io/-/commit/" + sha,
		},
		{
			name: "empty sha returns empty",
			u:    ghURL("https://github.com/scan-io-git/scan-io", ""),
			sha:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.u.CommitURL(tt.sha)
			if got != tt.want {
				t.Errorf("CommitURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPRURL(t *testing.T) {
	tests := []struct {
		name string
		u    *VCSURL
		want string
	}{
		{
			name: "bitbucket pr url",
			u:    bbDC("https://git.example-corp.internal/projects/SEC/repos/repo", "5"),
			want: "https://git.example-corp.internal/projects/SEC/repos/repo/pull-requests/5",
		},
		{
			name: "github pr url",
			u:    ghURL("https://github.com/scan-io-git/scan-io", "42"),
			want: "https://github.com/scan-io-git/scan-io/pull/42",
		},
		{
			name: "gitlab pr url",
			u:    glURL("https://gitlab.com/sec/scan-io", "7"),
			want: "https://gitlab.com/sec/scan-io/-/merge_requests/7",
		},
		{
			name: "no pr id returns empty",
			u:    ghURL("https://github.com/scan-io-git/scan-io", ""),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.u.PRURL()
			if got != tt.want {
				t.Errorf("PRURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLineAnchor(t *testing.T) {
	tests := []struct {
		name      string
		u         *VCSURL
		startLine int
		endLine   int
		want      string
	}{
		{"github single", ghURL("", ""), 10, 10, "#L10"},
		{"github range", ghURL("", ""), 10, 20, "#L10-L20"},
		{"github zero", ghURL("", ""), 0, 0, ""},
		{"gitlab single", glURL("", ""), 5, 5, "#L5"},
		{"gitlab range", glURL("", ""), 5, 15, "#L5-15"},
		{"bitbucket single", bbDC("", ""), 15, 15, "#15"},
		{"bitbucket range", bbDC("", ""), 15, 20, "#15-20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLineAnchor(tt.u.VCSType, tt.startLine, tt.endLine)
			if got != tt.want {
				t.Errorf("buildLineAnchor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInferVCSTypeFromShape(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    VCSType
		wantOK  bool
	}{
		{
			name:   "bitbucket scm clone path",
			rawURL: "https://git.example-corp.internal/scm/sec/repo.git",
			want:   Bitbucket,
			wantOK: true,
		},
		{
			name:   "bitbucket projects browse path",
			rawURL: "https://git.example-corp.internal/projects/SEC/repos/repo/browse/file.go",
			want:   Bitbucket,
			wantOK: true,
		},
		{
			name:   "bitbucket users repos path",
			rawURL: "https://git.corp.com/users/jdoe/repos/myrepo/browse",
			want:   Bitbucket,
			wantOK: true,
		},
		{
			name:   "bitbucket at query param",
			rawURL: "https://git.corp.com/browse/file.go?at=refs%2Fheads%2Fmain",
			want:   Bitbucket,
			wantOK: true,
		},
		{
			name:   "bitbucket ssh port 7999",
			rawURL: "ssh://git.corp.com:7999/sec/repo.git",
			want:   Bitbucket,
			wantOK: true,
		},
		{
			name:   "gitlab dash separator",
			rawURL: "https://git.corp.com/sec/repo/-/blob/main/file.go",
			want:   Gitlab,
			wantOK: true,
		},
		{
			name:   "unknown shape returns false",
			rawURL: "https://git.corp.com/sec/repo/blob/main/file.go",
			want:   0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.rawURL)
			if err != nil {
				t.Fatalf("url.Parse(%q): %v", tt.rawURL, err)
			}
			got, ok := inferVCSTypeFromShape(u)
			if ok != tt.wantOK {
				t.Errorf("inferVCSTypeFromShape() ok=%v, want %v", ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("inferVCSTypeFromShape() type=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWithOptions_HostMap(t *testing.T) {
	hostMap := map[string]VCSType{
		"git.example-corp.internal": Bitbucket,
	}

	// Host-map takes priority over hostname-substring detection.
	// Using an SCM clone URL; hostname doesn't contain "bitbucket" so
	// only the host map (or path inference) can resolve the type.
	u, err := ParseWithOptions("https://git.example-corp.internal/scm/sec/repo.git", ParseOptions{HostMap: hostMap})
	if err != nil {
		t.Fatalf("ParseWithOptions: %v", err)
	}
	if u.VCSType != Bitbucket {
		t.Errorf("VCSType = %v, want Bitbucket", u.VCSType)
	}
	if u.Namespace != "sec" {
		t.Errorf("Namespace = %q, want %q", u.Namespace, "sec")
	}
}

func TestParseWithOptions_ShapeDetection(t *testing.T) {
	// No config map, no flag — only path inference.
	u, err := ParseWithOptions("https://git.example-corp.internal/scm/sec/repo.git", ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions: %v", err)
	}
	if u.VCSType != Bitbucket {
		t.Errorf("VCSType = %v, want Bitbucket (detected via /scm/ path)", u.VCSType)
	}
}
