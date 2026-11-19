package sarif

import "testing"

func TestEnrichResultsFindingIDProperty(t *testing.T) {
	cases := []struct {
		name string
		raw  any
		want string
	}{
		{"plain id", "F-3f9a2c1d", "F-3f9a2c1d"},
		{"duplicate suffix", "F-3f9a2c1d.81d0", "F-3f9a2c1d.81d0"},
		{"surrounding space is trimmed", "  F-3f9a2c1d ", "F-3f9a2c1d"},
		{"any non-empty string is kept as given", "<b>F-1</b>", "<b>F-1</b>"},
		{"inner whitespace is kept", "F-3f9a 2c1d", "F-3f9a 2c1d"},
		{"long values are kept in full", "F-0123456789012345678901234567890123", "F-0123456789012345678901234567890123"},
		{"empty is rejected", "", ""},
		{"whitespace only is rejected", "  \t ", ""},
		{"non-string is rejected", 42, ""},
		{"absent", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := resultFor("rule.test")
			result.Properties = map[string]any{}
			if tc.raw != nil {
				result.Properties["findingId"] = tc.raw
			}
			report := makeSimpleReport("rule.test", nil, result)

			report.EnrichResultsFindingIDProperty()

			got, _ := result.Properties["FindingID"].(string)
			if got != tc.want {
				t.Errorf("FindingID = %q, want %q", got, tc.want)
			}
		})
	}
}
