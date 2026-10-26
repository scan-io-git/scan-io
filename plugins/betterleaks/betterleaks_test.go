package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/pkg/shared"
)

func newTestScanner() *ScannerBetterleaks {
	return newScannerBetterleaks(hclog.NewNullLogger())
}

func TestBuildCommandArgs(t *testing.T) {
	cases := []struct {
		name string
		req  shared.ScannerScanRequest
		want []string
	}{
		{
			name: "defaults to fs",
			req:  shared.ScannerScanRequest{TargetPath: "/src"},
			want: []string{"fs", "--exit-code", "2", "-o", "/out/r.json", "/src"},
		},
		{
			name: "git via command",
			req:  shared.ScannerScanRequest{Command: "git", TargetPath: "/repo"},
			want: []string{"git", "--exit-code", "2", "-o", "/out/r.json", "/repo"},
		},
		{
			name: "config path adds -c",
			req:  shared.ScannerScanRequest{TargetPath: "/src", ConfigPath: "/cfg.toml"},
			want: []string{"fs", "-c", "/cfg.toml", "--exit-code", "2", "-o", "/out/r.json", "/src"},
		},
		{
			name: "additional args keep order and come first",
			req: shared.ScannerScanRequest{
				TargetPath:     "/src",
				ConfigPath:     "/cfg.toml",
				AdditionalArgs: []string{"--no-allow-signatures", "--ignore-file", "/dev/null", "-j", "1", "--log-opts=--no-merges a..b"},
			},
			want: []string{
				"fs", "--no-allow-signatures", "--ignore-file", "/dev/null", "-j", "1", "--log-opts=--no-merges a..b",
				"-c", "/cfg.toml", "--exit-code", "2", "-o", "/out/r.json", "/src",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildCommandArgs(tc.req, "/out/r.json")

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("args:\n got %v\nwant %v", got, tc.want)
			}
		})
	}
}

func TestBuildCommandArgs_DoesNotMutateAdditionalArgs(t *testing.T) {
	extra := make([]string, 1, 10)
	extra[0] = "-j"
	req := shared.ScannerScanRequest{TargetPath: "/src", AdditionalArgs: extra}

	_ = buildCommandArgs(req, "/out/r.json")

	if len(req.AdditionalArgs) != 1 || req.AdditionalArgs[0] != "-j" {
		t.Errorf("AdditionalArgs mutated: %v", req.AdditionalArgs)
	}
}

func TestValidateScan_Command(t *testing.T) {
	target := t.TempDir()
	cases := []struct {
		command string
		wantErr bool
	}{
		{"", false},
		{"fs", false},
		{"git", false},
		{"url", true},
		{"filesystem", true},
		{"FS", true},
	}
	for _, tc := range cases {
		t.Run("command="+tc.command, func(t *testing.T) {
			req := &shared.ScannerScanRequest{TargetPath: target, ResultsPath: "/out/r.json", Command: tc.command}

			err := newTestScanner().validateScan(req)

			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestCheckReportFormat(t *testing.T) {
	cases := []struct {
		format         string
		wantReport     string
		wantConversion bool
	}{
		{"sarif", "json", true},
		{"json", "json", false},
		{"", "json", false},
		{"html", "json", false},
	}
	for _, tc := range cases {
		t.Run("format="+tc.format, func(t *testing.T) {
			req := &shared.ScannerScanRequest{ReportFormat: tc.format}

			orig, report, conv := newTestScanner().CheckReportFormat(req)

			if orig != tc.format || report != tc.wantReport || conv != tc.wantConversion {
				t.Errorf("got (%q, %q, %v)", orig, report, conv)
			}
		})
	}
}

func TestExitCodeOf(t *testing.T) {
	t.Run("nil error is exit 0", func(t *testing.T) {
		code, err := exitCodeOf(nil)
		if err != nil || code != 0 {
			t.Errorf("got (%d, %v)", code, err)
		}
	})
	t.Run("exit error carries the code", func(t *testing.T) {
		runErr := exec.Command("sh", "-c", "exit 2").Run()
		code, err := exitCodeOf(runErr)
		if err != nil || code != 2 {
			t.Errorf("got (%d, %v)", code, err)
		}
	})
	t.Run("start failure is an error", func(t *testing.T) {
		_, err := exitCodeOf(errors.New("exec: not found"))
		if err == nil {
			t.Error("want error")
		}
	})
}

func requireBetterleaks(t *testing.T) {
	t.Helper()
	bin := os.Getenv("BETTERLEAKS_BIN")
	if bin == "" {
		t.Skip("BETTERLEAKS_BIN is not set")
	}
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// Fake credential assembled at runtime so no secret-like literal sits in the repo.
func fakeToken() string {
	return "exk_" + strings.Repeat("aB3xY7", 6) + "aB3x"
}

const integrationConfig = `[extend]
useDefault = true

[[rules]]
id = "example-api-key"
description = "Example API key for tests."
regex = '''\b(exk_[A-Za-z0-9]{40})\b'''
keywords = ["exk_"]
confidence = "high"
tags = ["severity:high"]
`

func integrationArgs() []string {
	return []string{"--no-allow-signatures", "--ignore-file", "/dev/null", "--no-banner", "--no-color", "-j", "1"}
}

func readSarifResults(t *testing.T, path string) []any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Runs []struct {
			Results []any `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Runs[0].Results
}

func TestIntegration_FilesystemScanProducesSarif(t *testing.T) {
	requireBetterleaks(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "src")
	mustWrite(t, filepath.Join(target, "deploy.conf"), "key = "+fakeToken()+"\n")
	cfg := filepath.Join(dir, "bl.toml")
	mustWrite(t, cfg, integrationConfig)
	out := t.TempDir()

	resp, err := newTestScanner().Scan(shared.ScannerScanRequest{
		TargetPath:     target,
		ResultsPath:    filepath.Join(out, "result.sarif"),
		ConfigPath:     cfg,
		ReportFormat:   "sarif",
		AdditionalArgs: integrationArgs(),
	})

	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if resp.ResultsPath != filepath.Join(out, "result.sarif") {
		t.Errorf("results path: %q", resp.ResultsPath)
	}
	if got := readSarifResults(t, resp.ResultsPath); len(got) != 1 {
		t.Errorf("want 1 result, got %d", len(got))
	}
	if _, err := os.Stat(filepath.Join(out, "result.json")); err != nil {
		t.Errorf("raw json must be kept: %v", err)
	}
}

func TestIntegration_CleanScanProducesEmptySarif(t *testing.T) {
	requireBetterleaks(t)
	target := t.TempDir()
	mustWrite(t, filepath.Join(target, "readme.txt"), "nothing here\n")
	out := t.TempDir()

	resp, err := newTestScanner().Scan(shared.ScannerScanRequest{
		TargetPath:     target,
		ResultsPath:    filepath.Join(out, "result.sarif"),
		ReportFormat:   "sarif",
		AdditionalArgs: integrationArgs(),
	})

	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got := readSarifResults(t, resp.ResultsPath); len(got) != 0 {
		t.Errorf("want no results, got %d", len(got))
	}
}

func TestIntegration_GitScanProducesSarif(t *testing.T) {
	requireBetterleaks(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	cfg := filepath.Join(dir, "bl.toml")
	mustWrite(t, cfg, integrationConfig)
	repo := filepath.Join(dir, "repo")
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.email=fake@example.invalid", "-c", "user.name=Fake"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	git("init", "-q")
	mustWrite(t, filepath.Join(repo, "a.txt"), "hello\n")
	git("add", ".")
	git("commit", "-qm", "initial")
	mustWrite(t, filepath.Join(repo, "deploy.conf"), "key = "+fakeToken()+"\n")
	git("add", ".")
	git("commit", "-qm", "add config")
	out := t.TempDir()

	resp, err := newTestScanner().Scan(shared.ScannerScanRequest{
		Command:        "git",
		TargetPath:     repo,
		ResultsPath:    filepath.Join(out, "result.sarif"),
		ConfigPath:     cfg,
		ReportFormat:   "sarif",
		AdditionalArgs: append(integrationArgs(), "--log-opts=--no-merges HEAD~1..HEAD"),
	})

	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got := readSarifResults(t, resp.ResultsPath); len(got) != 1 {
		t.Errorf("want 1 result, got %d", len(got))
	}
}

func TestIntegration_ErrorCases(t *testing.T) {
	requireBetterleaks(t)
	target := t.TempDir()
	mustWrite(t, filepath.Join(target, "a.txt"), "x\n")
	badCfg := filepath.Join(t.TempDir(), "bad.toml")
	mustWrite(t, badCfg, "not [valid toml\n")
	cases := []struct {
		name string
		req  shared.ScannerScanRequest
	}{
		{"broken config", shared.ScannerScanRequest{ConfigPath: badCfg, AdditionalArgs: integrationArgs()}},
		{"missing ignore file", shared.ScannerScanRequest{AdditionalArgs: []string{"--ignore-file", "/nonexistent/ignore", "-j", "1"}}},
		{"unknown flag", shared.ScannerScanRequest{AdditionalArgs: []string{"--no-such-flag"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := t.TempDir()
			tc.req.TargetPath = target
			tc.req.ResultsPath = filepath.Join(out, "result.sarif")
			tc.req.ReportFormat = "sarif"

			_, err := newTestScanner().Scan(tc.req)

			if err == nil {
				t.Fatal("want error")
			}
			if _, statErr := os.Stat(filepath.Join(out, "result.sarif")); statErr == nil {
				t.Error("no SARIF must be written on error")
			}
		})
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
