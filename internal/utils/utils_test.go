package common

import (
	"testing"
)

func TestGetDomain(t *testing.T) {
	var tests = []struct {
		repoURL string
		want    string
	}{
		{"https://github.com/juice-shop/juice-shop.git", "github.com"},
		{"git@github.com:juice-shop/juice-shop.git", "github.com"},
		{"http://bitbucket.example.com:7994/scm/thet/projectsource.git", "bitbucket.example.com"},
		{"ssh://git@bitbucket.example.com:7999/scm/thet/projectsource.git", "bitbucket.example.com"},
		{"https://gitlab.com/juice-shop/juice-shop.git", "gitlab.com"},
		{"git@gitlab.com:juice-shop/juice-shop.git", "gitlab.com"},
	}
	for _, tt := range tests {
		t.Run(tt.repoURL, func(t *testing.T) {
			got, err := GetDomain(tt.repoURL)
			if err != nil {
				t.Error(err)
			}
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetPath(t *testing.T) {
	var tests = []struct {
		repoURL string
		want    string
	}{
		{"https://github.com/juice-shop/juice-shop.git", "juice-shop/juice-shop"},
		{"git@github.com:juice-shop/juice-shop.git", "juice-shop/juice-shop"},
		{"http://bitbucket.example.com:7994/scm/thet/projectsource.git", "scm/thet/projectsource"},
		{"ssh://git@bitbucket.example.com:7999/scm/thet/projectsource.git", "scm/thet/projectsource"},
	}
	for _, tt := range tests {
		t.Run(tt.repoURL, func(t *testing.T) {
			got, err := GetPath(tt.repoURL)
			if err != nil {
				t.Error(err)
			}
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSplitPathOnNamespaceAndRepoName(t *testing.T) {
	var tests = []struct {
		repoURL string
		want    []string
	}{
		{"juice-shop/juice-shop", []string{"juice-shop", "juice-shop"}},
		{"juice-shop/juice-shop", []string{"juice-shop", "juice-shop"}},
		{"scm/thet/projectsource", []string{"scm/thet", "projectsource"}},
		{"scm/thet/projectsource", []string{"scm/thet", "projectsource"}},
	}
	for _, tt := range tests {
		t.Run(tt.repoURL, func(t *testing.T) {
			ns, repo := SplitPathOnNamespaceAndRepoName(tt.repoURL)
			if ns != tt.want[0] {
				t.Errorf("got %s, want %s", ns, tt.want[0])
			}
			if repo != tt.want[1] {
				t.Errorf("got %s, want %s", repo, tt.want[1])
			}
		})
	}
}
