package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTempJSON(t *testing.T, issues []*Trufflehog3Issue) string {
	t.Helper()
	data, err := json.Marshal(issues)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	f, err := os.CreateTemp(t.TempDir(), "th3-*.json")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	return f.Name()
}

func readSARIF(t *testing.T, jsonPath string) map[string]interface{} {
	t.Helper()
	sarifPath := jsonPath[:len(jsonPath)-len(filepath.Ext(jsonPath))] + ".sarif"
	data, err := os.ReadFile(sarifPath)
	if err != nil {
		t.Fatalf("read sarif: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal sarif: %v", err)
	}
	return out
}

func firstResult(t *testing.T, sarif map[string]interface{}) map[string]interface{} {
	t.Helper()
	runs := sarif["runs"].([]interface{})
	results := runs[0].(map[string]interface{})["results"].([]interface{})
	return results[0].(map[string]interface{})
}

func TestJsonToSarifReport_EmbedSecretInSnippet(t *testing.T) {
	issues := []*Trufflehog3Issue{
		{
			Rule:   &Trufflehog3Rule{ID: "high-entropy", Message: "High-entropy string", Severity: "MEDIUM"},
			Path:   "config/settings.yaml",
			Line:   "42",
			Secret: "xoxp-000111222333-000111222333-abcdef1234567890abcdef123456",
			ID:     "8835dfe9-7e77-3399-8805-8c38972edb4a",
		},
	}

	jsonPath := writeTempJSON(t, issues)
	if _, err := JsonToSarifReport(jsonPath); err != nil {
		t.Fatalf("JsonToSarifReport: %v", err)
	}

	sarif := readSARIF(t, jsonPath)
	result := firstResult(t, sarif)

	// Verify region.snippet.text contains the secret.
	locations := result["locations"].([]interface{})
	physLoc := locations[0].(map[string]interface{})["physicalLocation"].(map[string]interface{})
	region := physLoc["region"].(map[string]interface{})
	snippet, ok := region["snippet"].(map[string]interface{})
	if !ok {
		t.Fatal("region.snippet not present in SARIF output")
	}
	if got := snippet["text"].(string); got != issues[0].Secret {
		t.Errorf("region.snippet.text = %q, want %q", got, issues[0].Secret)
	}

	// Verify result.fingerprints["th3/v1"] contains the UUID.
	fingerprints, ok := result["fingerprints"].(map[string]interface{})
	if !ok {
		t.Fatal("result.fingerprints not present in SARIF output")
	}
	if got := fingerprints["th3/v1"].(string); got != issues[0].ID {
		t.Errorf("fingerprints[th3/v1] = %q, want %q", got, issues[0].ID)
	}
}

func TestJsonToSarifReport_EmptySecretAndID(t *testing.T) {
	issues := []*Trufflehog3Issue{
		{
			Rule:   &Trufflehog3Rule{ID: "high-entropy", Message: "High-entropy string", Severity: "MEDIUM"},
			Path:   "src/main.go",
			Line:   "10",
			Secret: "",
			ID:     "",
		},
	}

	jsonPath := writeTempJSON(t, issues)
	if _, err := JsonToSarifReport(jsonPath); err != nil {
		t.Fatalf("JsonToSarifReport: %v", err)
	}

	sarif := readSARIF(t, jsonPath)
	result := firstResult(t, sarif)

	// snippet and fingerprints should be absent when Secret and ID are empty.
	locations := result["locations"].([]interface{})
	physLoc := locations[0].(map[string]interface{})["physicalLocation"].(map[string]interface{})
	region := physLoc["region"].(map[string]interface{})
	if _, ok := region["snippet"]; ok {
		t.Error("region.snippet present but Secret was empty")
	}
	if _, ok := result["fingerprints"]; ok {
		t.Error("result.fingerprints present but ID was empty")
	}
}

func TestJsonToSarifReport_ContextFieldParsed(t *testing.T) {
	issues := []*Trufflehog3Issue{
		{
			Rule:    &Trufflehog3Rule{ID: "high-entropy", Message: "msg", Severity: "MEDIUM"},
			Path:    "README.md",
			Line:    "5",
			Secret:  "abc123",
			Context: map[string]string{"5": "| secret | abc123 |"},
		},
	}

	// Round-trip through JSON to confirm Context is parsed correctly.
	data, _ := json.Marshal(issues)
	var parsed Trufflehog3Report
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := parsed[0].Context["5"]; got != issues[0].Context["5"] {
		t.Errorf("Context[5] = %q, want %q", got, issues[0].Context["5"])
	}
}
