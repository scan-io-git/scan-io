package git

import (
	"testing"
)

func TestSameRemote_Equivalent(t *testing.T) {
	cases := [][2]string{
		{"https://github.com/org/repo.git", "https://github.com/org/repo"},
		{"https://token@github.com/org/repo", "https://github.com/org/repo"},
		{"git@github.com:org/repo.git", "https://github.com/org/repo"},
		{"ssh://git@github.com/org/repo", "git@github.com:org/repo.git"},
		{"HTTPS://GitHub.com/Org/Repo", "git@github.com:org/repo"},
		{"https://github.com/org/repo/?ref=main#frag", "https://github.com/org/repo"},
	}

	for i, c := range cases {
		got := sameRemote(c[0], c[1])
		if !got {
			t.Fatalf("case %d expected true, got false: %q vs %q", i, c[0], c[1])
		}
	}
}

func TestSameRemote_Different(t *testing.T) {
	cases := [][2]string{
		{"https://github.com/org/repo", "https://github.com/org/repo2"},
		{"https://github.com/org/repo", "https://github-enterprise.local/org/repo"},
		{"git@github.com:org/repo", "git@github.com:other/repo"},
		{"https://github.com/org/repo", "https://gitlab.com/org/repo"},
		{"https://github.com/org/repo", "https://github.com/other/repo"},
	}

	for i, c := range cases {
		if sameRemote(c[0], c[1]) {
			t.Fatalf("case %d expected false, got true: %q vs %q", i, c[0], c[1])
		}
	}
}

func TestNormalizeRemote(t *testing.T) {
	host, path, err := normalizeRemote("git@github.com:Org/Repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "github.com" || path != "org/repo" {
		t.Fatalf("unexpected normalize result: host=%q path=%q", host, path)
	}

	host, path, err = normalizeRemote("ssh://git@github.com/Org/Repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "github.com" || path != "org/repo" {
		t.Fatalf("unexpected normalize result: host=%q path=%q", host, path)
	}
}
