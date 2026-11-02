package internal

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sarifDoc = map[string]any

func loadFixture(t *testing.T, name string) *Report {
	t.Helper()
	report, err := ParseReport(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return report
}

func parseJSONReport(t *testing.T, raw string) *Report {
	t.Helper()
	var report Report
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	return &report
}

func toSarifBytes(t *testing.T, report *Report) []byte {
	t.Helper()
	out, err := JSONToSarif(report, "")
	if err != nil {
		t.Fatalf("JSONToSarif: %v", err)
	}
	var buf bytes.Buffer
	if err := out.PrettyWrite(&buf); err != nil {
		t.Fatalf("PrettyWrite: %v", err)
	}
	return buf.Bytes()
}

func toSarifDoc(t *testing.T, report *Report) sarifDoc {
	t.Helper()
	var doc sarifDoc
	if err := json.Unmarshal(toSarifBytes(t, report), &doc); err != nil {
		t.Fatalf("unmarshal sarif: %v", err)
	}
	return doc
}

func asMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("want object, got %T (%v)", v, v)
	}
	return m
}

func asSlice(t *testing.T, v any) []any {
	t.Helper()
	s, ok := v.([]any)
	if !ok {
		t.Fatalf("want array, got %T (%v)", v, v)
	}
	return s
}

func runOf(t *testing.T, doc sarifDoc) map[string]any {
	t.Helper()
	return asMap(t, asSlice(t, doc["runs"])[0])
}

func resultsOf(t *testing.T, doc sarifDoc) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, r := range asSlice(t, runOf(t, doc)["results"]) {
		out = append(out, asMap(t, r))
	}
	return out
}

func resultByRule(t *testing.T, doc sarifDoc, ruleID string) map[string]any {
	t.Helper()
	for _, r := range resultsOf(t, doc) {
		if r["ruleId"] == ruleID {
			return r
		}
	}
	t.Fatalf("no result for rule %q", ruleID)
	return nil
}

func driverOf(t *testing.T, doc sarifDoc) map[string]any {
	t.Helper()
	return asMap(t, asMap(t, runOf(t, doc)["tool"])["driver"])
}

func physicalOf(t *testing.T, result map[string]any) map[string]any {
	t.Helper()
	locs := asSlice(t, result["locations"])
	return asMap(t, asMap(t, locs[0])["physicalLocation"])
}

func TestJSONToSarif_Driver(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "fs.json"))

	driver := driverOf(t, doc)

	cases := map[string]string{
		"name":           "betterleaks",
		"version":        "2.0.0-rc.2",
		"informationUri": "https://github.com/betterleaks/betterleaks",
	}
	for key, want := range cases {
		if driver[key] != want {
			t.Errorf("driver.%s: want %q, got %v", key, want, driver[key])
		}
	}
	if doc["version"] != "2.1.0" {
		t.Errorf("SARIF version: want 2.1.0, got %v", doc["version"])
	}
}

func TestJSONToSarif_RulesAreDistinctWithDescriptionAndTags(t *testing.T) {
	report := loadFixture(t, "fs.json")
	report.Findings = append(report.Findings, report.Findings[1])
	doc := toSarifDoc(t, report)

	rules := asSlice(t, driverOf(t, doc)["rules"])

	if len(rules) != 2 {
		t.Fatalf("want 2 distinct rules for 3 findings, got %d", len(rules))
	}
	rule := asMap(t, rules[1])
	if rule["id"] != "example-api-key" {
		t.Errorf("rule id: got %v", rule["id"])
	}
	wantDesc := "Example API key (exk_ prefix) used for test fixtures."
	for _, key := range []string{"shortDescription", "fullDescription"} {
		if got := asMap(t, rule[key])["text"]; got != wantDesc {
			t.Errorf("%s.text: want %q, got %v", key, wantDesc, got)
		}
	}
	tags := asSlice(t, asMap(t, rule["properties"])["tags"])
	if len(tags) != 2 || tags[0] != "example" || tags[1] != "severity:high" {
		t.Errorf("rule tags: got %v", tags)
	}
}

func TestJSONToSarif_LevelFromSeverityTag(t *testing.T) {
	cases := []struct {
		name       string
		tags       string
		confidence string
		want       string
	}{
		{"critical is error", `["severity:critical"]`, "low", "error"},
		{"high is error", `["example","severity:high"]`, "low", "error"},
		{"medium is warning", `["severity:medium"]`, "high", "warning"},
		{"low is note", `["severity:low"]`, "high", "note"},
		{"no severity tag defaults to error", `["example"]`, "low", "error"},
		{"no tags defaults to error", `null`, "medium", "error"},
		{"unknown severity defaults to error", `["severity:bogus"]`, "high", "error"},
		{"severity is case insensitive", `["severity:Medium"]`, "high", "warning"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := parseJSONReport(t, `{"findings":[{"rule_id":"r","description":"d","confidence":"`+
				tc.confidence+`","match":{"full":"x","fingerprint":"f"},"location":{"path":"a","start_line":1,"end_line":1,"start_column":1,"end_column":1},"tags":`+
				tc.tags+`}],"scan":{"state":"complete"}}`)

			got := resultByRule(t, toSarifDoc(t, report), "r")["level"]

			if got != tc.want {
				t.Errorf("level: want %q, got %v", tc.want, got)
			}
		})
	}
}

func TestJSONToSarif_ResultFieldsFromFilesystemFixture(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "fs.json"))

	result := resultByRule(t, doc, "example-api-key")

	if got := asMap(t, result["message"])["text"]; got != "Example API key (exk_ prefix) used for test fixtures." {
		t.Errorf("message.text: got %v", got)
	}
	if result["level"] != "error" {
		t.Errorf("level: got %v", result["level"])
	}
	props := asMap(t, result["properties"])
	if props["confidence"] != "high" {
		t.Errorf("confidence: got %v", props["confidence"])
	}
	if props["resource"] != "fs.content" {
		t.Errorf("resource: got %v", props["resource"])
	}
	fp := asMap(t, result["partialFingerprints"])
	if fp["betterleaks/v1"] != "335e68a155ed4aad69e2dd13f9b0dd0538d23028ffbb3e578d10740c978d9bc8" {
		t.Errorf("fingerprint: got %v", fp["betterleaks/v1"])
	}
}

func TestJSONToSarif_ConfidenceIsNotUsedForLevel(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "fs.json"))

	result := resultByRule(t, doc, "generic-password")

	if asMap(t, result["properties"])["confidence"] != "low" {
		t.Fatalf("fixture precondition: want low confidence")
	}
	if result["level"] != "error" {
		t.Errorf("low-confidence finding without severity tag: want level error, got %v", result["level"])
	}
}

func TestJSONToSarif_RegionAndSnippet(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "fs.json"))

	phys := physicalOf(t, resultByRule(t, doc, "generic-password"))

	if got := asMap(t, phys["artifactLocation"])["uri"]; got != "fs/settings.yml" {
		t.Errorf("uri: got %v", got)
	}
	region := asMap(t, phys["region"])
	want := map[string]float64{"startLine": 2, "endLine": 2, "startColumn": 3, "endColumn": 32}
	for key, w := range want {
		if region[key] != w {
			t.Errorf("region.%s: want %v, got %v", key, w, region[key])
		}
	}
	if got := asMap(t, region["snippet"])["text"]; got != `password = "Hunter2Hunter2xQ"` {
		t.Errorf("snippet.text: got %v", got)
	}
}

func TestJSONToSarif_StripsLeadingDotSlashFromURI(t *testing.T) {
	cases := []struct{ path, want string }{
		{"./app/ams.py", "app/ams.py"},
		{"app/settings.yml", "app/settings.yml"},
		{"./", ""},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			report := parseJSONReport(t, `{"findings":[{"rule_id":"r","description":"d","confidence":"high","match":{"full":"x"},"location":{"path":"`+
				tc.path+`","start_line":1,"end_line":1,"start_column":1,"end_column":1}}],"scan":{"state":"complete"}}`)

			result := resultByRule(t, toSarifDoc(t, report), "r")

			if tc.want == "" {
				return
			}
			if got := asMap(t, physicalOf(t, result)["artifactLocation"])["uri"]; got != tc.want {
				t.Errorf("uri: want %q, got %v", tc.want, got)
			}
		})
	}
}

func TestJSONToSarif_GitFindingProperties(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "git.json"))

	props := asMap(t, resultByRule(t, doc, "generic-password")["properties"])

	if props["resource"] != "git.patch_content" {
		t.Errorf("resource: got %v", props["resource"])
	}
	if props["git.sha"] != "6f0cb8913c6dc9620fe14b6c1896f7a20bc28984" {
		t.Errorf("git.sha: got %v", props["git.sha"])
	}
	if props["git.date"] != "2026-10-20T05:26:41Z" {
		t.Errorf("git.date: got %v", props["git.date"])
	}
}

func TestJSONToSarif_CommitMessageFindingHasNoLocation(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "git.json"))

	result := resultByRule(t, doc, "example-api-key")

	if _, ok := result["locations"]; ok {
		t.Errorf("commit message finding must have no locations, got %v", result["locations"])
	}
	props := asMap(t, result["properties"])
	if props["resource"] != "git.commit_message" {
		t.Errorf("resource: got %v", props["resource"])
	}
	if props["git.sha"] != "6f0cb8913c6dc9620fe14b6c1896f7a20bc28984" {
		t.Errorf("git.sha: got %v", props["git.sha"])
	}
}

func TestJSONToSarif_EncodingsDepthAndComponentRules(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "fs_components_encoded.json"))

	t.Run("encodings and decode depth", func(t *testing.T) {
		var encoded map[string]any
		for _, r := range resultsOf(t, doc) {
			if _, ok := asMap(t, r["properties"])["encodings"]; ok {
				encoded = asMap(t, r["properties"])
			}
		}
		if encoded == nil {
			t.Fatal("no result with encodings")
		}
		if enc := asSlice(t, encoded["encodings"]); len(enc) != 1 || enc[0] != "base64" {
			t.Errorf("encodings: got %v", enc)
		}
		if encoded["decode_depth"] != float64(1) {
			t.Errorf("decode_depth: got %v", encoded["decode_depth"])
		}
	})

	t.Run("component rule ids", func(t *testing.T) {
		var found []any
		for _, r := range resultsOf(t, doc) {
			if ids, ok := asMap(t, r["properties"])["component_rule_ids"]; ok {
				found = asSlice(t, ids)
			}
		}
		if len(found) != 1 || found[0] != "example-api-host" {
			t.Errorf("component_rule_ids: got %v", found)
		}
	})
}

func TestJSONToSarif_ValidationStatusReasonAndMetadata(t *testing.T) {
	report := parseJSONReport(t, `{"findings":[{"rule_id":"r","description":"d","confidence":"high","match":{"full":"x"},
		"location":{"path":"a","start_line":1,"end_line":1,"start_column":1,"end_column":1},
		"analysis":{"status":"valid","status_reason":"provider said so",
			"status_metadata":{"account":"sample-account","scopes":"read,write"}}}],"scan":{"state":"complete"}}`)

	props := asMap(t, resultByRule(t, toSarifDoc(t, report), "r")["properties"])

	if props["validation_status"] != "valid" {
		t.Errorf("validation_status: got %v", props["validation_status"])
	}
	if props["validation_reason"] != "provider said so" {
		t.Errorf("validation_reason: got %v", props["validation_reason"])
	}
	meta := asMap(t, props["validation_metadata"])
	if meta["account"] != "sample-account" || meta["scopes"] != "read,write" || len(meta) != 2 {
		t.Errorf("validation_metadata: got %v", meta)
	}
}

func TestJSONToSarif_InvocationAndRunProperties(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "fs.json"))

	run := runOf(t, doc)

	inv := asMap(t, asSlice(t, run["invocations"])[0])
	if inv["executionSuccessful"] != true {
		t.Errorf("executionSuccessful: got %v", inv["executionSuccessful"])
	}
	props := asMap(t, run["properties"])
	if props["config_hash"] != "3a5bafd24d9a1bf5b85c59f2d19b680f71b00e18012855e65271f8e43344896f" {
		t.Errorf("config_hash: got %v", props["config_hash"])
	}
	if props["bytes_scanned"] != float64(105) {
		t.Errorf("bytes_scanned: got %v", props["bytes_scanned"])
	}
	if props["num_findings"] != float64(2) {
		t.Errorf("num_findings: got %v", props["num_findings"])
	}
}

func TestJSONToSarif_EmptyFindingsGiveEmptyResults(t *testing.T) {
	report := parseJSONReport(t, `{"schema_version":"1","findings":[],"scan":{"state":"complete","betterleaks_version":"2.0.0-rc.2"}}`)

	doc := toSarifDoc(t, report)

	if got := asSlice(t, runOf(t, doc)["results"]); len(got) != 0 {
		t.Errorf("want no results, got %d", len(got))
	}
	if driverOf(t, doc)["name"] != "betterleaks" {
		t.Errorf("driver name: got %v", driverOf(t, doc)["name"])
	}
}

func TestJSONToSarif_SecretCapturesAndContext(t *testing.T) {
	report := parseJSONReport(t, `{"findings":[{"rule_id":"r","description":"d","confidence":"high",
		"match":{"full":"password: Sample-Value-1","value":"Sample-Value-1","fingerprint":"fp1",
			"captures":{"scheme":"https","password":"Sample-Value-1"},
			"context":"line before\npassword: Sample-Value-1\nline after"},
		"location":{"path":"a.yml","start_line":2,"end_line":2,"start_column":1,"end_column":24}}],
		"scan":{"state":"complete"}}`)

	result := resultByRule(t, toSarifDoc(t, report), "r")

	props := asMap(t, result["properties"])
	if props["secret"] != "Sample-Value-1" {
		t.Errorf("secret: got %v", props["secret"])
	}
	captures := asMap(t, props["captures"])
	if captures["scheme"] != "https" || captures["password"] != "Sample-Value-1" || len(captures) != 2 {
		t.Errorf("captures: got %v", captures)
	}
	physical := physicalOf(t, result)
	ctx := asMap(t, asMap(t, physical["contextRegion"])["snippet"])
	if ctx["text"] != "line before\npassword: Sample-Value-1\nline after" {
		t.Errorf("contextRegion snippet: got %v", ctx["text"])
	}
	region := asMap(t, physical["region"])
	if asMap(t, region["snippet"])["text"] != "password: Sample-Value-1" || region["startLine"] != float64(2) {
		t.Errorf("region must stay unchanged, got %v", region)
	}
}

func TestJSONToSarif_OmitsEmptySecretCapturesAndContext(t *testing.T) {
	doc := toSarifDoc(t, parseJSONReport(t, `{"findings":[{"rule_id":"r","description":"d","confidence":"high",
		"match":{"full":"x"},"location":{"path":"a","start_line":1,"end_line":1,"start_column":1,"end_column":1}}],
		"scan":{"state":"complete"}}`))
	result := resultByRule(t, doc, "r")

	props := asMap(t, result["properties"])
	for _, key := range []string{"secret", "captures", "components", "git.message", "validation_reason", "validation_metadata"} {
		if _, ok := props[key]; ok {
			t.Errorf("property %q must be absent, got %v", key, props[key])
		}
	}
	if _, ok := physicalOf(t, result)["contextRegion"]; ok {
		t.Error("contextRegion must be absent without match.context")
	}
}

func TestJSONToSarif_ComponentsFromSeveralSets(t *testing.T) {
	report := parseJSONReport(t, `{"findings":[{"rule_id":"r","description":"d","confidence":"high",
		"match":{"full":"pw=Sample-Value-2","value":"Sample-Value-2"},
		"location":{"path":"a","start_line":3,"end_line":3,"start_column":1,"end_column":17},
		"component_sets":[
		 {"components":[{"rule_id":"user-rule","optional":false,
		   "match":{"full":"user=sample-user","value":"sample-user"},
		   "location":{"path":"a","start_line":2,"end_line":2,"start_column":1,"end_column":16}}]},
		 {"components":[
		   {"rule_id":"host-rule","optional":true,"match":{"full":" host.example.test","value":"host.example.test"},
		    "location":{"path":"b","start_line":1,"end_line":1,"start_column":5,"end_column":22}},
		   {"rule_id":"user-rule","match":{"value":"other-user"}}]}]}],
		"scan":{"state":"complete"}}`)

	props := asMap(t, resultByRule(t, toSarifDoc(t, report), "r")["properties"])

	comps := asSlice(t, props["components"])
	if len(comps) != 3 {
		t.Fatalf("components: want 3, got %d (%v)", len(comps), comps)
	}
	first := asMap(t, comps[0])
	if first["set"] != float64(0) || first["rule_id"] != "user-rule" || first["value"] != "sample-user" ||
		first["full"] != "user=sample-user" || first["optional"] != false {
		t.Errorf("first component: got %v", first)
	}
	if loc := asMap(t, first["location"]); loc["path"] != "a" || loc["start_line"] != float64(2) || loc["end_column"] != float64(16) {
		t.Errorf("first component location: got %v", loc)
	}
	second := asMap(t, comps[1])
	if second["set"] != float64(1) || second["rule_id"] != "host-rule" || second["optional"] != true {
		t.Errorf("second component: got %v", second)
	}
	if third := asMap(t, comps[2]); third["set"] != float64(1) || third["value"] != "other-user" {
		t.Errorf("third component: got %v", third)
	}
	ids := asSlice(t, props["component_rule_ids"])
	if len(ids) != 2 || ids[0] != "user-rule" || ids[1] != "host-rule" {
		t.Errorf("component_rule_ids: got %v", ids)
	}
}

func TestJSONToSarif_ComponentsAndSecretOfEncodedFinding(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "fs_components_encoded.json"))

	for _, r := range resultsOf(t, doc) {
		props := asMap(t, r["properties"])
		if props["secret"] != "exk_Qm7ZpL2vXc9RtB4nYh8WdK3sGf6JaE1uNo5TiV0x" {
			t.Errorf("secret: got %v", props["secret"])
		}
		if _, encoded := props["encodings"]; encoded {
			if _, ok := props["components"]; ok {
				t.Errorf("encoded finding has no components, got %v", props["components"])
			}
			continue
		}
		comps := asSlice(t, props["components"])
		if len(comps) != 1 {
			t.Fatalf("components: want 1, got %v", comps)
		}
		c := asMap(t, comps[0])
		if c["rule_id"] != "example-api-host" || c["value"] != "api.example.com" ||
			c["full"] != " api.example.com" || c["optional"] != true {
			t.Errorf("component: got %v", c)
		}
		if loc := asMap(t, c["location"]); loc["path"] != "fs2/comp.txt" || loc["start_line"] != float64(1) {
			t.Errorf("component location: got %v", loc)
		}
	}
}

func TestJSONToSarif_GitMessage(t *testing.T) {
	doc := toSarifDoc(t, loadFixture(t, "git.json"))

	for _, rule := range []string{"generic-password", "example-api-key"} {
		props := asMap(t, resultByRule(t, doc, rule)["properties"])
		if props["git.message"] != "add config with exk_Qm7ZpL2vXc9RtB4nYh8WdK3sGf6JaE1uNo5TiV0x" {
			t.Errorf("%s git.message: got %v", rule, props["git.message"])
		}
	}
}

func TestJSONToSarif_DoesNotCopyAuthorIdentity(t *testing.T) {
	out := string(toSarifBytes(t, loadFixture(t, "git.json")))

	for _, s := range []string{"dev@example.com", "Example Dev", "author_email", "committer"} {
		if strings.Contains(out, s) {
			t.Errorf("SARIF output contains %q", s)
		}
	}
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestLoadCompleteReport(t *testing.T) {
	complete := `{"findings":[],"scan":{"state":"complete"}}`
	incomplete := `{"findings":[{"rule_id":"r"}],"scan":{"state":"incomplete"}}`
	cases := []struct {
		name     string
		exitCode int
		content  string // empty means the file is not created
		wantErr  bool
	}{
		{"exit 0 complete is ok", 0, complete, false},
		{"exit 2 complete is ok", 2, complete, false},
		{"exit 1 is an error", 1, complete, true},
		{"exit 126 is an error", 126, complete, true},
		{"exit -1 is an error", -1, complete, true},
		{"incomplete state with exit 0 is an error", 0, incomplete, true},
		{"incomplete state with exit 2 is an error", 2, incomplete, true},
		{"missing state is an error", 0, `{"findings":[],"scan":{}}`, true},
		{"missing report is an error", 0, "", true},
		{"broken json is an error", 2, `{"findings": [`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "report.json")
			if tc.content != "" {
				path = writeFile(t, "report.json", tc.content)
			}

			report, err := LoadCompleteReport(tc.exitCode, path)

			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr && report != nil {
				t.Errorf("partial report must not be returned on error")
			}
		})
	}
}

func TestLoadCompleteReport_RealIncompleteFixtureIsRejected(t *testing.T) {
	_, err := LoadCompleteReport(0, filepath.Join("..", "testdata", "incomplete.json"))

	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Errorf("want incomplete state error, got %v", err)
	}
}

func TestWriteSarif_WritesFileNextToJSONAndKeepsJSON(t *testing.T) {
	jsonPath := writeFile(t, "out.json", `{"findings":[],"scan":{"state":"complete"}}`)
	report, err := ParseReport(jsonPath)
	if err != nil {
		t.Fatal(err)
	}

	sarifPath, err := WriteSarif(report, jsonPath, "")

	if err != nil {
		t.Fatalf("WriteSarif: %v", err)
	}
	if sarifPath != strings.TrimSuffix(jsonPath, ".json")+".sarif" {
		t.Errorf("sarif path: got %q", sarifPath)
	}
	for _, p := range []string{jsonPath, sarifPath} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
}

func TestArtifactURI(t *testing.T) {
	tests := []struct {
		name, path, target, want string
	}{
		{"absolute inside target", "/work/repo/pkg/a.go", "/work/repo", "pkg/a.go"},
		{"absolute inside target with trailing slash", "/work/repo/pkg/a.go", "/work/repo/", "pkg/a.go"},
		{"relative git path unchanged", "pkg/a.go", "/work/repo", "pkg/a.go"},
		{"dot slash prefix trimmed", "./pkg/a.go", "/work/repo", "pkg/a.go"},
		{"absolute outside target unchanged", "/other/pkg/a.go", "/work/repo", "/other/pkg/a.go"},
		{"sibling with shared prefix unchanged", "/work/repo2/a.go", "/work/repo", "/work/repo2/a.go"},
		{"target is the file itself", "/work/repo/a.go", "/work/repo/a.go", "a.go"},
		{"empty target unchanged", "/work/repo/a.go", "", "/work/repo/a.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := artifactURI(tt.path, tt.target); got != tt.want {
				t.Errorf("artifactURI(%q, %q) = %q, want %q", tt.path, tt.target, got, tt.want)
			}
		})
	}
}

func TestJSONToSarif_ComponentPathIsRelativeToTarget(t *testing.T) {
	report := parseJSONReport(t, `{"findings":[{"rule_id":"r","description":"d","confidence":"high",
		"match":{"full":"x"},"location":{"path":"/src/a.txt","start_line":2,"end_line":2,"start_column":1,"end_column":1},
		"component_sets":[{"components":[{"rule_id":"c","location":{"path":"/src/a.txt","start_line":1,"end_line":1,"start_column":1,"end_column":1}}]}]}],
		"scan":{"state":"complete"}}`)
	out, err := JSONToSarif(report, "/src")
	if err != nil {
		t.Fatalf("JSONToSarif: %v", err)
	}
	var buf bytes.Buffer
	if err := out.PrettyWrite(&buf); err != nil {
		t.Fatalf("PrettyWrite: %v", err)
	}
	var doc sarifDoc
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal sarif: %v", err)
	}

	props := asMap(t, resultByRule(t, doc, "r")["properties"])

	loc := asMap(t, asMap(t, asSlice(t, props["components"])[0])["location"])
	if loc["path"] != "a.txt" {
		t.Errorf("component path: got %v", loc["path"])
	}
}

func TestJSONToSarif_NonStringCaptureValuesAreKept(t *testing.T) {
	report := parseJSONReport(t, `{"findings":[{"rule_id":"r","description":"d","confidence":"high",
		"match":{"full":"x","captures":{"port":8080,"empty":null,"name":"sample"}},
		"location":{"path":"a","start_line":1,"end_line":1,"start_column":1,"end_column":1}}],
		"scan":{"state":"complete"}}`)

	props := asMap(t, resultByRule(t, toSarifDoc(t, report), "r")["properties"])

	captures := asMap(t, props["captures"])
	if captures["port"] != float64(8080) || captures["name"] != "sample" {
		t.Errorf("captures: got %v", captures)
	}
	if v, ok := captures["empty"]; !ok || v != nil {
		t.Errorf("null capture must be kept, got %v (present=%v)", v, ok)
	}
}
