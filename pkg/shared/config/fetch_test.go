package config

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestValidateFetchConfig(t *testing.T) {
	tests := []struct {
		name    string
		in      *[]string
		want    []string
		wantErr string
	}{
		{name: "unset list gets the defaults", in: nil, want: DefaultDiffCopyPaths},
		{name: "empty list stays empty", in: &[]string{}, want: []string{}},
		{name: "custom list replaces the defaults", in: &[]string{".git", "ci/scanner.yaml"}, want: []string{".git", "ci/scanner.yaml"}},
		{name: "entries are cleaned", in: &[]string{"./.tool.yml"}, want: []string{".tool.yml"}},
		{name: "absolute path is rejected", in: &[]string{"/etc/placeholder"}, wantErr: "/etc/placeholder"},
		{name: "parent path is rejected", in: &[]string{"../outside"}, wantErr: "../outside"},
		{name: "path escaping through a folder is rejected", in: &[]string{"a/../../b"}, wantErr: "a/../../b"},
		{name: "repository root is rejected", in: &[]string{"."}, wantErr: `"."`},
		{name: "empty entry is rejected", in: &[]string{""}, wantErr: `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Fetch{DiffCopyPaths: tt.in}
			err := ValidateFetchConfig(&f)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want one naming %s", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := f.CopyPaths(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyPaths() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestValidateFetchConfigDoesNotShareDefaults(t *testing.T) {
	before := append([]string(nil), DefaultDiffCopyPaths...)
	f := Fetch{}
	if err := ValidateFetchConfig(&f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.CopyPaths()[0] = "changed"
	if !reflect.DeepEqual(DefaultDiffCopyPaths, before) {
		t.Errorf("DefaultDiffCopyPaths changed to %v", DefaultDiffCopyPaths)
	}
}

func TestFetchCopyPaths(t *testing.T) {
	if got := (Fetch{}).CopyPaths(); got != nil {
		t.Errorf("unset list: CopyPaths() = %#v, want nil", got)
	}
	if got := (Fetch{DiffCopyPaths: &[]string{".git"}}).CopyPaths(); !reflect.DeepEqual(got, []string{".git"}) {
		t.Errorf("CopyPaths() = %#v, want [.git]", got)
	}
}

func TestFetchConfigFromYAML(t *testing.T) {
	tests := []struct {
		name      string
		doc       string
		wantUnset bool
	}{
		{name: "empty list is kept as set", doc: "fetch:\n  diff_copy_paths: []\n"},
		{name: "missing section is unset", doc: "logger:\n  level: info\n", wantUnset: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			if err := yaml.Unmarshal([]byte(tt.doc), &cfg); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if gotUnset := cfg.Fetch.DiffCopyPaths == nil; gotUnset != tt.wantUnset {
				t.Fatalf("DiffCopyPaths unset = %v, want %v", gotUnset, tt.wantUnset)
			}
			if !tt.wantUnset && len(*cfg.Fetch.DiffCopyPaths) != 0 {
				t.Errorf("DiffCopyPaths = %v, want empty", *cfg.Fetch.DiffCopyPaths)
			}
		})
	}
}
