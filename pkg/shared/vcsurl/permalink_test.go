package vcsurl

import (
	"errors"
	"testing"
)

func TestBuildPermalink(t *testing.T) {
	tests := []struct {
		name        string
		params      PermalinkParams
		expected    string
		expectedErr error
	}{
		// GitHub tests
		{
			name: "GitHub with line range",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
				File:      "src/app.go",
				StartLine: 10,
				EndLine:   20,
			},
			expected:    "https://github.com/org/repo/blob/main/src/app.go#L10-L20",
			expectedErr: nil,
		},
		{
			name: "GitHub with single line",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Project:   "repo",
				Ref:       "abc123",
				File:      "main.go",
				StartLine: 5,
				EndLine:   5,
			},
			expected:    "https://github.com/org/repo/blob/abc123/main.go#L5",
			expectedErr: nil,
		},
		{
			name: "GitHub without line numbers",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
				File:      "README.md",
			},
			expected:    "https://github.com/org/repo/blob/main/README.md",
			expectedErr: nil,
		},
		{
			name: "GitHub self-hosted",
			params: PermalinkParams{
				VCSType:   Github,
				Host:      "github.example.com",
				Namespace: "team",
				Project:   "app",
				Ref:       "develop",
				File:      "cmd/main.go",
				StartLine: 100,
			},
			expected:    "https://github.example.com/team/app/blob/develop/cmd/main.go#L100",
			expectedErr: nil,
		},
		{
			name: "GitHub with nested namespace",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org/team",
				Project:   "repo",
				Ref:       "main",
				File:      "app.go",
				StartLine: 1,
			},
			expected:    "https://github.com/org/team/repo/blob/main/app.go#L1",
			expectedErr: nil,
		},

		// GitLab tests
		{
			name: "GitLab with line range",
			params: PermalinkParams{
				VCSType:   Gitlab,
				Namespace: "group/subgroup",
				Project:   "project",
				Ref:       "main",
				File:      "src/main.py",
				StartLine: 10,
				EndLine:   25,
			},
			expected:    "https://gitlab.com/group/subgroup/project/-/blob/main/src/main.py#L10-25",
			expectedErr: nil,
		},
		{
			name: "GitLab with single line",
			params: PermalinkParams{
				VCSType:   Gitlab,
				Namespace: "org",
				Project:   "repo",
				Ref:       "feature-branch",
				File:      "lib/utils.rb",
				StartLine: 42,
			},
			expected:    "https://gitlab.com/org/repo/-/blob/feature-branch/lib/utils.rb#L42",
			expectedErr: nil,
		},
		{
			name: "GitLab self-hosted",
			params: PermalinkParams{
				VCSType:   Gitlab,
				Host:      "gitlab.internal.company.com",
				Namespace: "team",
				Project:   "service",
				Ref:       "v1.2.3",
				File:      "pkg/handler.go",
				StartLine: 50,
				EndLine:   75,
			},
			expected:    "https://gitlab.internal.company.com/team/service/-/blob/v1.2.3/pkg/handler.go#L50-75",
			expectedErr: nil,
		},

		// Bitbucket tests
		{
			name: "Bitbucket with line range",
			params: PermalinkParams{
				VCSType:   Bitbucket,
				Host:      "bitbucket.example.com",
				Namespace: "PROJECT",
				Project:   "repo",
				Ref:       "abc123def",
				File:      "src/main/App.java",
				StartLine: 100,
				EndLine:   150,
			},
			expected:    "https://bitbucket.example.com/projects/PROJECT/repos/repo/browse/src/main/App.java?at=abc123def#100-150",
			expectedErr: nil,
		},
		{
			name: "Bitbucket with single line",
			params: PermalinkParams{
				VCSType:   Bitbucket,
				Host:      "bitbucket.company.com",
				Namespace: "PROJ",
				Project:   "api",
				Ref:       "main",
				File:      "handler.go",
				StartLine: 25,
			},
			expected:    "https://bitbucket.company.com/projects/PROJ/repos/api/browse/handler.go?at=main#25",
			expectedErr: nil,
		},
		{
			name: "Bitbucket Cloud default host",
			params: PermalinkParams{
				VCSType:   Bitbucket,
				Namespace: "workspace",
				Project:   "repo",
				Ref:       "main",
				File:      "app.py",
				StartLine: 10,
			},
			expected:    "https://bitbucket.org/projects/workspace/repos/repo/browse/app.py?at=main#10",
			expectedErr: nil,
		},

		// Generic VCS tests
		{
			name: "Generic VCS uses GitHub format",
			params: PermalinkParams{
				VCSType:   GenericVCS,
				Host:      "git.example.com",
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
				File:      "app.go",
				StartLine: 10,
				EndLine:   20,
			},
			expected:    "https://git.example.com/org/repo/blob/main/app.go#L10-L20",
			expectedErr: nil,
		},
		{
			name: "UnknownVCS uses GitHub format",
			params: PermalinkParams{
				VCSType:   UnknownVCS,
				Host:      "vcs.example.com",
				Namespace: "team",
				Project:   "app",
				Ref:       "develop",
				File:      "main.go",
				StartLine: 5,
			},
			expected:    "https://vcs.example.com/team/app/blob/develop/main.go#L5",
			expectedErr: nil,
		},

		// Edge cases
		{
			name: "File path with leading slash",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
				File:      "/src/app.go",
				StartLine: 1,
			},
			expected:    "https://github.com/org/repo/blob/main/src/app.go#L1",
			expectedErr: nil,
		},
		{
			name: "File path with backslashes (Windows)",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
				File:      "src\\subfolder\\app.go",
				StartLine: 1,
			},
			expected:    "https://github.com/org/repo/blob/main/src/subfolder/app.go#L1",
			expectedErr: nil,
		},
		{
			name: "EndLine less than StartLine treated as single line",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
				File:      "app.go",
				StartLine: 10,
				EndLine:   5,
			},
			expected:    "https://github.com/org/repo/blob/main/app.go#L10",
			expectedErr: nil,
		},
		{
			name: "EndLine zero treated as single line",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
				File:      "app.go",
				StartLine: 10,
				EndLine:   0,
			},
			expected:    "https://github.com/org/repo/blob/main/app.go#L10",
			expectedErr: nil,
		},

		// Error cases - missing required parameters
		{
			name: "Missing namespace",
			params: PermalinkParams{
				VCSType: Github,
				Project: "repo",
				Ref:     "main",
				File:    "app.go",
			},
			expected:    "",
			expectedErr: ErrMissingNamespace,
		},
		{
			name: "Missing project",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Ref:       "main",
				File:      "app.go",
			},
			expected:    "",
			expectedErr: ErrMissingProject,
		},
		{
			name: "Missing ref",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Project:   "repo",
				File:      "app.go",
			},
			expected:    "",
			expectedErr: ErrMissingRef,
		},
		{
			name: "Missing file",
			params: PermalinkParams{
				VCSType:   Github,
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
			},
			expected:    "",
			expectedErr: ErrMissingFile,
		},
		{
			name: "Generic VCS without host",
			params: PermalinkParams{
				VCSType:   GenericVCS,
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
				File:      "app.go",
			},
			expected:    "",
			expectedErr: ErrMissingHost,
		},
		{
			name: "UnknownVCS without host",
			params: PermalinkParams{
				VCSType:   UnknownVCS,
				Namespace: "org",
				Project:   "repo",
				Ref:       "main",
				File:      "app.go",
			},
			expected:    "",
			expectedErr: ErrMissingHost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildPermalink(tt.params)

			// Check error
			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("BuildPermalink() expected error %v, got nil", tt.expectedErr)
					return
				}
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("BuildPermalink() error = %v, expected %v", err, tt.expectedErr)
					return
				}
			} else {
				if err != nil {
					t.Errorf("BuildPermalink() unexpected error: %v", err)
					return
				}
			}

			// Check result
			if result != tt.expected {
				t.Errorf("BuildPermalink() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestBuildPermalinkErrorMessages(t *testing.T) {
	// Verify error messages are descriptive
	tests := []struct {
		name           string
		params         PermalinkParams
		expectedErr    error
		errMsgContains string
	}{
		{
			name:           "Missing namespace error message",
			params:         PermalinkParams{VCSType: Github, Project: "repo", Ref: "main", File: "app.go"},
			expectedErr:    ErrMissingNamespace,
			errMsgContains: "namespace",
		},
		{
			name:           "Missing project error message",
			params:         PermalinkParams{VCSType: Github, Namespace: "org", Ref: "main", File: "app.go"},
			expectedErr:    ErrMissingProject,
			errMsgContains: "project",
		},
		{
			name:           "Missing ref error message",
			params:         PermalinkParams{VCSType: Github, Namespace: "org", Project: "repo", File: "app.go"},
			expectedErr:    ErrMissingRef,
			errMsgContains: "ref",
		},
		{
			name:           "Missing file error message",
			params:         PermalinkParams{VCSType: Github, Namespace: "org", Project: "repo", Ref: "main"},
			expectedErr:    ErrMissingFile,
			errMsgContains: "file",
		},
		{
			name:           "Missing host error message",
			params:         PermalinkParams{VCSType: GenericVCS, Namespace: "org", Project: "repo", Ref: "main", File: "app.go"},
			expectedErr:    ErrMissingHost,
			errMsgContains: "host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildPermalink(tt.params)
			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestBuildLineAnchor(t *testing.T) {
	tests := []struct {
		name      string
		vcsType   VCSType
		startLine int
		endLine   int
		expected  string
	}{
		// GitHub anchors
		{"GitHub single line", Github, 10, 10, "#L10"},
		{"GitHub line range", Github, 10, 20, "#L10-L20"},
		{"GitHub no line", Github, 0, 0, ""},
		{"GitHub negative start", Github, -1, 10, ""},

		// GitLab anchors
		{"GitLab single line", Gitlab, 42, 42, "#L42"},
		{"GitLab line range", Gitlab, 10, 25, "#L10-25"},
		{"GitLab no line", Gitlab, 0, 0, ""},

		// Bitbucket anchors
		{"Bitbucket single line", Bitbucket, 100, 100, "#100"},
		{"Bitbucket line range", Bitbucket, 50, 75, "#50-75"},
		{"Bitbucket no line", Bitbucket, 0, 0, ""},

		// Generic uses GitHub format
		{"Generic single line", GenericVCS, 5, 5, "#L5"},
		{"Generic line range", GenericVCS, 1, 10, "#L1-L10"},

		// Edge cases
		{"EndLine less than StartLine", Github, 20, 10, "#L20"},
		{"EndLine zero", Github, 15, 0, "#L15"},
		{"StartLine only", Github, 7, 0, "#L7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildLineAnchor(tt.vcsType, tt.startLine, tt.endLine)
			if result != tt.expected {
				t.Errorf("buildLineAnchor(%v, %d, %d) = %q, expected %q",
					tt.vcsType, tt.startLine, tt.endLine, result, tt.expected)
			}
		})
	}
}
