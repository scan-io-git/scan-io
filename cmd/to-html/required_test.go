package tohtml

import (
	"strings"
	"testing"
)

func TestParseRequiredPolicy(t *testing.T) {
	tests := []struct {
		name        string
		flag        string
		env         map[string]string
		wantEnabled bool
		wantErr     string // substring expected in the error message, "" means no error
		wantBlocker map[string]bool
		wantThr     map[string]float64 // only keys to assert
		neverDemote string             // --never-demote flag value
		wantPinned  map[string]bool    // expected NeverDemoteSeverities
	}{
		{
			name: "disabled when nothing set", flag: "", env: nil, wantEnabled: false,
		},
		{
			name: "flag severities without threshold have empty thresholds", flag: "critical,high", wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true},
			wantThr:     map[string]float64{}, // no threshold configured → confidence filtering disabled
		},
		{
			name: "flag with threshold override", flag: "critical:0.50,high:0.90", wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true},
			wantThr:     map[string]float64{"critical": 0.5, "high": 0.9},
		},
		{
			name: "flag mixed with and without thresholds", flag: "critical,high:0.90,medium", wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true, "medium": true},
			wantThr:     map[string]float64{"high": 0.9},
		},
		{
			name: "flag beats env", flag: "low", env: map[string]string{"SCANIO_BLOCKER_SEVERITIES": "critical"},
			wantEnabled: true, wantBlocker: map[string]bool{"low": true},
		},
		{
			name: "env fallback when flag empty", flag: "",
			env:         map[string]string{"SCANIO_BLOCKER_SEVERITIES": "critical,high", "SCANIO_CONFIDENCE_THRESHOLD_HIGH": "0.95"},
			wantEnabled: true, wantBlocker: map[string]bool{"critical": true, "high": true},
			wantThr: map[string]float64{"high": 0.95}, // critical has no threshold env var → no confidence filtering for it
		},
		{
			name: "case insensitive severities", flag: "CRITICAL,High", wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true},
		},
		{
			name: "malformed threshold via flag", flag: "high:oops",
			wantErr: `invalid confidence threshold "oops" for severity "high"`,
		},
		{
			name:    "malformed threshold via env",
			env:     map[string]string{"SCANIO_BLOCKER_SEVERITIES": "high", "SCANIO_CONFIDENCE_THRESHOLD_HIGH": "oops"},
			wantErr: `invalid confidence threshold "oops" for SCANIO_CONFIDENCE_THRESHOLD_HIGH`,
		},
		{
			name: "out of range threshold above 1.0 via flag", flag: "high:1.5",
			wantErr: `invalid confidence threshold "1.5" for severity "high"`,
		},
		{
			name: "out of range threshold below 0.0 via flag", flag: "high:-0.1",
			wantErr: `invalid confidence threshold "-0.1" for severity "high"`,
		},
		{
			name:    "out of range threshold via env",
			env:     map[string]string{"SCANIO_BLOCKER_SEVERITIES": "high", "SCANIO_CONFIDENCE_THRESHOLD_HIGH": "2"},
			wantErr: `invalid confidence threshold "2" for SCANIO_CONFIDENCE_THRESHOLD_HIGH`,
		},
		{
			name: "boundary thresholds 0.0 and 1.0 are valid", flag: "critical:0.0,high:1.0", wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true},
			wantThr:     map[string]float64{"critical": 0.0, "high": 1.0},
		},
		{
			name: "unknown severity via flag", flag: "hihg",
			wantErr: `invalid severity "hihg" in --required`,
		},
		{
			name:    "unknown severity via env",
			env:     map[string]string{"SCANIO_BLOCKER_SEVERITIES": "hihg"},
			wantErr: `invalid severity "hihg" in SCANIO_BLOCKER_SEVERITIES`,
		},
		{
			name: "never-demote via flag", flag: "critical,high", neverDemote: "critical", wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true},
			wantThr:     map[string]float64{},
			wantPinned:  map[string]bool{"critical": true, "high": false},
		},
		{
			name: "never-demote via env", flag: "critical,high", env: map[string]string{"SCANIO_NEVER_DEMOTE": "critical"},
			wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true},
			wantThr:     map[string]float64{},
			wantPinned:  map[string]bool{"critical": true},
		},
		{
			name: "never-demote flag beats env", flag: "critical,high", neverDemote: "high",
			env:         map[string]string{"SCANIO_NEVER_DEMOTE": "critical"},
			wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true},
			wantThr:     map[string]float64{},
			wantPinned:  map[string]bool{"high": true, "critical": false},
		},
		{
			name: "never-demote alone does not enable classification", flag: "", neverDemote: "critical",
			wantEnabled: false,
		},
		{
			name: "unknown severity in never-demote flag", flag: "critical", neverDemote: "crticial",
			wantErr: `invalid severity "crticial" in --never-demote`,
		},
		{
			name: "unknown severity in never-demote env", flag: "critical",
			env:     map[string]string{"SCANIO_NEVER_DEMOTE": "hihg"},
			wantErr: `invalid severity "hihg" in SCANIO_NEVER_DEMOTE`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SCANIO_BLOCKER_SEVERITIES", "")
			t.Setenv("SCANIO_NEVER_DEMOTE", "")
			t.Setenv("SCANIO_CONFIDENCE_THRESHOLD_CRITICAL", "")
			t.Setenv("SCANIO_CONFIDENCE_THRESHOLD_HIGH", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			policy, enabled, err := parseRequiredPolicy(tt.flag, tt.neverDemote)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("err = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %q, want to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if enabled != tt.wantEnabled {
				t.Fatalf("enabled = %v, want %v", enabled, tt.wantEnabled)
			}
			if !enabled {
				return
			}
			for k, v := range tt.wantBlocker {
				if policy.BlockerSeverities[k] != v {
					t.Errorf("blocker[%q] = %v, want %v", k, policy.BlockerSeverities[k], v)
				}
			}
			for k, v := range tt.wantPinned {
				if policy.NeverDemoteSeverities[k] != v {
					t.Errorf("neverDemote[%q] = %v, want %v", k, policy.NeverDemoteSeverities[k], v)
				}
			}
			if tt.wantPinned == nil && len(policy.NeverDemoteSeverities) != 0 {
				t.Errorf("neverDemote = %v, want empty", policy.NeverDemoteSeverities)
			}
			if tt.wantThr != nil && len(policy.Thresholds) != len(tt.wantThr) {
				t.Errorf("thresholds len = %d, want %d: %v", len(policy.Thresholds), len(tt.wantThr), policy.Thresholds)
			}
			for k, v := range tt.wantThr {
				if policy.Thresholds[k] != v {
					t.Errorf("threshold[%q] = %v, want %v", k, policy.Thresholds[k], v)
				}
			}
		})
	}
}
