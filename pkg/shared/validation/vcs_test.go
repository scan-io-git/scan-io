package validation

import (
	"testing"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

func newBaseRequest() shared.VCSRequestBase {
	return shared.VCSRequestBase{
		VCSDomain: "bitbucket.example.com",
		RepoParam: shared.RepositoryParams{
			Domain:        "bitbucket.example.com",
			Namespace:     "team",
			Repository:    "repo",
			PullRequestID: "42",
		},
		Action: "addInLineComments",
	}
}

func TestValidateAddInLineCommentsListArgs_NoContent(t *testing.T) {
	req := &shared.VCSAddInLineCommentsListRequest{VCSRequestBase: newBaseRequest()}

	if err := ValidateAddInLineCommentsListArgs(req); err == nil || err.Error() != "no comments to post" {
		t.Fatalf("expected no comments error, got %v", err)
	}
}

func TestValidateAddInLineCommentsListArgs_AllCommentsEmpty(t *testing.T) {
	req := &shared.VCSAddInLineCommentsListRequest{
		VCSRequestBase: newBaseRequest(),
		Comments: []shared.Comment{
			{Body: "   "},
		},
	}

	if err := ValidateAddInLineCommentsListArgs(req); err == nil || err.Error() != "no comments to post" {
		t.Fatalf("expected no comments error, got %v", err)
	}
}

func TestValidateAddInLineCommentsListArgs_SummaryOnly(t *testing.T) {
	req := &shared.VCSAddInLineCommentsListRequest{
		VCSRequestBase: newBaseRequest(),
		Summary:        "  summary provided  ",
	}

	if err := ValidateAddInLineCommentsListArgs(req); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestValidateAddInLineCommentsListArgs_WithComment(t *testing.T) {
	req := &shared.VCSAddInLineCommentsListRequest{
		VCSRequestBase: newBaseRequest(),
		Comments: []shared.Comment{
			{Body: "issue body"},
			{Body: "   "},
		},
	}

	if err := ValidateAddInLineCommentsListArgs(req); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestValidateAddInLineCommentsListArgs_MissingBaseField(t *testing.T) {
	req := &shared.VCSAddInLineCommentsListRequest{
		VCSRequestBase: shared.VCSRequestBase{},
		Comments:       []shared.Comment{{Body: "hello"}},
	}

	if err := ValidateAddInLineCommentsListArgs(req); err == nil {
		t.Fatal("expected base validation error")
	}
}

func TestValidateAddCommentToPRArgs_TrimsBody(t *testing.T) {
	req := &shared.VCSAddCommentToPRRequest{
		VCSRequestBase: newBaseRequest(),
		Comment:        shared.Comment{Body: "   "},
	}

	if err := ValidateAddCommentToPRArgs(req); err == nil || err.Error() != "comment is required" {
		t.Fatalf("expected comment required error, got %v", err)
	}

	req.Comment.Body = " ok "
	if err := ValidateAddCommentToPRArgs(req); err != nil {
		t.Fatalf("expected trimmed comment to pass, got %v", err)
	}
}
