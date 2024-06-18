package template

import (
	"fmt"
	"html/template"
	"time"
)

// add adds two integers and returns the result.
// helper function for html template
func add(a, b int) int {
	return a + b
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

func NewTemplate(templateFile string) (*template.Template, error) {
	return template.New("report.html").
		Funcs(template.FuncMap{
			"add":              add,
			"generateSequence": generateSequence,
			"formatDateTime":   formatDateTime,
		}).
		ParseFiles(templateFile)
}
