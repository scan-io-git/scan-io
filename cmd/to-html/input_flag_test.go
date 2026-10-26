package tohtml

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/pflag"
)

// newInputFlagSet mirrors the production registration of the --input/-i flag
// (StringArrayVarP, not StringSliceVarP) against a fresh, isolated FlagSet so
// each test case parses independently.
func newInputFlagSet(inputs *[]string) *pflag.FlagSet {
	fs := pflag.NewFlagSet("to-html", pflag.ContinueOnError)
	fs.StringArrayVarP(inputs, "input", "i", nil, "test")
	return fs
}

func TestInputFlag_OneValue(t *testing.T) {
	var inputs []string
	fs := newInputFlagSet(&inputs)
	if err := fs.Parse([]string{"-i", "a.sarif"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"a.sarif"}
	if !reflect.DeepEqual(inputs, want) {
		t.Errorf("Inputs = %v, want %v", inputs, want)
	}
}

func TestInputFlag_SeveralValues(t *testing.T) {
	var inputs []string
	fs := newInputFlagSet(&inputs)
	if err := fs.Parse([]string{"--input", "a.sarif", "--input", "b.sarif", "--input", "c.sarif"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"a.sarif", "b.sarif", "c.sarif"}
	if !reflect.DeepEqual(inputs, want) {
		t.Errorf("Inputs = %v, want %v", inputs, want)
	}
}

func TestInputFlag_RepeatedShorthand(t *testing.T) {
	var inputs []string
	fs := newInputFlagSet(&inputs)
	if err := fs.Parse([]string{"-i", "a.sarif", "-i", "b.sarif"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"a.sarif", "b.sarif"}
	if !reflect.DeepEqual(inputs, want) {
		t.Errorf("Inputs = %v, want %v", inputs, want)
	}
}

// TestInputFlag_CommaNotSplit guards the "array, not slice" decision: a StringSlice
// flag would split a comma-bearing path into two bogus inputs.
func TestInputFlag_CommaNotSplit(t *testing.T) {
	var inputs []string
	fs := newInputFlagSet(&inputs)
	if err := fs.Parse([]string{"-i", "path/with,comma.sarif"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"path/with,comma.sarif"}
	if !reflect.DeepEqual(inputs, want) {
		t.Errorf("Inputs = %v, want %v (StringArray must not comma-split)", inputs, want)
	}
}

// resetToHTMLOptionsForTest points allToHTMLOptions at a scratch output file inside
// dir and returns it, so RunE hard-fail tests can assert no partial file is left
// behind.
func resetToHTMLOptionsForTest(t *testing.T, dir string, inputs []string) string {
	t.Helper()
	Init(nil, hclog.NewNullLogger())
	output := filepath.Join(dir, "report.html")
	allToHTMLOptions = ToHTMLOptions{
		Inputs:       inputs,
		OutputFile:   output,
		SourceFolder: dir,
	}
	return output
}

func TestRunE_HardFailsOnMissingInput(t *testing.T) {
	dir := t.TempDir()
	output := resetToHTMLOptionsForTest(t, dir, []string{filepath.Join(dir, "does-not-exist.sarif")})

	if err := ToHtmlCmd.RunE(ToHtmlCmd, nil); err == nil {
		t.Fatal("RunE() = nil error, want error for a missing input path")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output file must not be written on hard fail, stat err = %v", err)
	}
}

func TestRunE_HardFailsOnMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	badInput := filepath.Join(dir, "bad.sarif")
	if err := os.WriteFile(badInput, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	output := resetToHTMLOptionsForTest(t, dir, []string{badInput})

	if err := ToHtmlCmd.RunE(ToHtmlCmd, nil); err == nil {
		t.Fatal("RunE() = nil error, want error for malformed sarif JSON")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output file must not be written on hard fail, stat err = %v", err)
	}
}

func TestRunE_HardFailsOnNoInputs(t *testing.T) {
	dir := t.TempDir()
	output := resetToHTMLOptionsForTest(t, dir, nil)

	if err := ToHtmlCmd.RunE(ToHtmlCmd, nil); err == nil {
		t.Fatal("RunE() = nil error, want error when no --input is given")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output file must not be written on hard fail, stat err = %v", err)
	}
}
