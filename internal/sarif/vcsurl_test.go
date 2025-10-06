package sarif

import (
	"testing"

	"github.com/owenrumney/go-sarif/v2/sarif"

	"github.com/scan-io-git/scan-io/internal/git"
)

func TestNewLocationURLBuilder_IncludesRepositorySubfolder(t *testing.T) {
	commit := "deadbeefcafebabe"
	repo := "https://github.com/acme/monorepo"
	meta := &git.RepositoryMetadata{
		CommitHash:         &commit,
		RepositoryFullName: &repo,
		Subfolder:          "/services/mail",
	}

	builder, err := NewLocationURLBuilder(meta, "github")
	if err != nil {
		t.Fatalf("builder error: %v", err)
	}

	artifact := "src/app.js"
	start := 42
	location := &sarif.Location{
		PhysicalLocation: &sarif.PhysicalLocation{
			ArtifactLocation: &sarif.ArtifactLocation{URI: &artifact},
			Region:           &sarif.Region{StartLine: &start},
		},
	}

	url := builder(location)
	want := "https://github.com/acme/monorepo/blob/deadbeefcafebabe/services/mail/src/app.js#L42"
	if url != want {
		t.Fatalf("unexpected url\nwant: %s\n got: %s", want, url)
	}
}

func TestNewLocationURLBuilder_NoSubfolder(t *testing.T) {
	commit := "0123456789abcdef"
	repo := "https://bitbucket.org/projects/proj/repos/mono"
	meta := &git.RepositoryMetadata{
		CommitHash:         &commit,
		RepositoryFullName: &repo,
		Subfolder:          "",
	}

	builder, err := NewLocationURLBuilder(meta, "bitbucket")
	if err != nil {
		t.Fatalf("builder error: %v", err)
	}

	artifact := "pkg/service/main.go"
	start := 7
	location := &sarif.Location{
		PhysicalLocation: &sarif.PhysicalLocation{
			ArtifactLocation: &sarif.ArtifactLocation{URI: &artifact},
			Region:           &sarif.Region{StartLine: &start},
		},
	}

	url := builder(location)
	want := "https://bitbucket.org/projects/proj/repos/mono/browse/pkg/service/main.go?at=0123456789abcdef#7"
	if url != want {
		t.Fatalf("unexpected url\nwant: %s\n got: %s", want, url)
	}
}
