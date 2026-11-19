package sarif

import "strings"

// EnrichResultsFindingIDProperty projects properties.findingId, the stable
// display ID a producer assigns (for example "F-3f9a2c1d", or "F-3f9a2c1d.81d0"
// for a duplicate), into Properties["FindingID"]. The value is taken as given
// when it is a string that is not empty after trimming; the report truncates
// long values visually and escapes them on output. Results without a usable ID
// get no property and keep their positional number.
func (r Report) EnrichResultsFindingIDProperty() {
	for _, run := range r.Runs {
		for _, result := range run.Results {
			if result == nil || result.Properties == nil {
				continue
			}
			id, _ := result.Properties["findingId"].(string)
			if id = strings.TrimSpace(id); id != "" {
				result.Properties["FindingID"] = id
			}
		}
	}
}
