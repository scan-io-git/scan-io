package main

import (
	"strings"
	"testing"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

func TestFilterIssuesByBody(t *testing.T) {
	tests := []struct {
		name       string
		bodyFilter string
		issues     []shared.IssueParams
		expected   int
	}{
		{
			name:       "no filter - return all issues",
			bodyFilter: "",
			issues: []shared.IssueParams{
				{Number: 1, Body: "Regular issue"},
				{Number: 2, Body: "Scanio managed issue\n> [!NOTE]\n> This issue was created and will be managed by scanio automation"},
				{Number: 3, Body: "Another regular issue"},
			},
			expected: 3,
		},
		{
			name:       "filter for scanio managed issues",
			bodyFilter: "> [!NOTE]\n> This issue was created and will be managed by scanio automation",
			issues: []shared.IssueParams{
				{Number: 1, Body: "Regular issue"},
				{Number: 2, Body: "Scanio managed issue\n> [!NOTE]\n> This issue was created and will be managed by scanio automation"},
				{Number: 3, Body: "Another regular issue"},
			},
			expected: 1,
		},
		{
			name:       "filter with partial match",
			bodyFilter: "scanio automation",
			issues: []shared.IssueParams{
				{Number: 1, Body: "Regular issue"},
				{Number: 2, Body: "Scanio managed issue\n> [!NOTE]\n> This issue was created and will be managed by scanio automation"},
				{Number: 3, Body: "Another regular issue"},
			},
			expected: 1,
		},
		{
			name:       "filter with no matches",
			bodyFilter: "nonexistent text",
			issues: []shared.IssueParams{
				{Number: 1, Body: "Regular issue"},
				{Number: 2, Body: "Another regular issue"},
			},
			expected: 0,
		},
		{
			name:       "filter with multiple matches",
			bodyFilter: "issue",
			issues: []shared.IssueParams{
				{Number: 1, Body: "Regular issue"},
				{Number: 2, Body: "Another issue"},
				{Number: 3, Body: "Not a problem"},
			},
			expected: 2,
		},
		{
			name:       "case sensitive filtering",
			bodyFilter: "Issue",
			issues: []shared.IssueParams{
				{Number: 1, Body: "Regular issue"},
				{Number: 2, Body: "Another Issue"},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterIssuesByBody(tt.issues, tt.bodyFilter)

			if len(result) != tt.expected {
				t.Errorf("expected %d issues, got %d", tt.expected, len(result))
			}

			// Verify that the correct issues are returned
			if tt.bodyFilter != "" {
				for _, issue := range result {
					if !strings.Contains(issue.Body, tt.bodyFilter) {
						t.Errorf("issue %d does not contain filter text: %s", issue.Number, tt.bodyFilter)
					}
				}
			}
		})
	}
}
