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
