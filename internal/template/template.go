package template

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// derefStr dereferences a *string, returning "" for nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// add adds two integers and returns the result.
// helper function for html template
func add(a, b int) int {
	return a + b
}

// stripFragment removes the fragment portion (#...) from a URL string.
// helper function for html template
func stripFragment(url string) string {
	before, _, _ := strings.Cut(url, "#")
	return before
}

// lineAnchor returns the VCS-dialect-correct URL fragment for a single line number.
// vcsType is the string form ("bitbucket", "gitlab", "github", "generic", or "").
// Bitbucket DC uses bare "#N"; GitLab uses "#LN"; GitHub/others use "#LN".
func lineAnchor(vcsType string, line int) string {
	if line <= 0 {
		return ""
	}
	if vcsType == "bitbucket" {
		return fmt.Sprintf("#%d", line)
	}
	return fmt.Sprintf("#L%d", line)
}

// generateSequence generates a slice of integers from 1 to n.
// helper function for html template
func generateSequence(n int) []int {
	var sequence []int
	for i := 1; i <= n; i++ {
		sequence = append(sequence, i)
	}
	return sequence
}

// ordinalDate returns a string with the ordinal number of the day
// helper function for html template
func ordinalDate(day int) string {
	suffix := "th"
	switch day {
	case 1, 21, 31:
		suffix = "st"
	case 2, 22:
		suffix = "nd"
	case 3, 23:
		suffix = "rd"
	}
	return fmt.Sprintf("%d%s", day, suffix)
}

// formatDateTime formats a time.Time object into the specified string format.
// helper function for html template
func formatDateTime(t time.Time) string {
	day := ordinalDate(t.Day())
	return fmt.Sprintf("%s %s %d %d:%02d:%02d %s", day, t.Month(), t.Year(), t.Hour()%12, t.Minute(), t.Second(), t.Format("pm"))
}

func NewTemplate(templateFile string, options ...func(*template.Template)) (*template.Template, error) {
	t := template.New("report.html").
		Funcs(template.FuncMap{
			"add":              add,
			"derefStr":         derefStr,
			"generateSequence": generateSequence,
			"formatDateTime":   formatDateTime,
			"stripFragment":    stripFragment,
			"lineAnchor":       lineAnchor,
		})
	for _, o := range options {
		o(t)
	}
	return t.ParseFiles(templateFile)
}

func WithFuncs(funcs template.FuncMap) func(*template.Template) {
	return func(t *template.Template) {
		t.Funcs(funcs)
	}
}
