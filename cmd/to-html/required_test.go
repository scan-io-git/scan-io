package tohtml

import "testing"

func TestParseRequiredPolicy(t *testing.T) {
	tests := []struct {
		name        string
		flag        string
		env         map[string]string
		wantEnabled bool
		wantBlocker map[string]bool
		wantThr     map[string]float64 // only keys to assert
	}{
		{
			name: "disabled when nothing set", flag: "", env: nil, wantEnabled: false,
		},
		{
			name: "flag severities only use defaults", flag: "critical,high", wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true},
			wantThr:     map[string]float64{"high": 0.6},
		},
		{
			name: "flag with threshold override", flag: "critical:0.50,high:0.90", wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true},
			wantThr:     map[string]float64{"critical": 0.5, "high": 0.9},
		},
		{
			name: "flag beats env", flag: "low", env: map[string]string{"SCANIO_BLOCKER_SEVERITIES": "critical"},
			wantEnabled: true, wantBlocker: map[string]bool{"low": true},
		},
		{
			name: "env fallback when flag empty", flag: "",
			env:         map[string]string{"SCANIO_BLOCKER_SEVERITIES": "critical,high", "SCANIO_CONFIDENCE_THRESHOLD_HIGH": "0.95"},
			wantEnabled: true, wantBlocker: map[string]bool{"critical": true, "high": true},
			wantThr: map[string]float64{"high": 0.95, "critical": 0.5},
		},
		{
			name: "case insensitive severities", flag: "CRITICAL,High", wantEnabled: true,
			wantBlocker: map[string]bool{"critical": true, "high": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SCANIO_BLOCKER_SEVERITIES", "")
			t.Setenv("SCANIO_CONFIDENCE_THRESHOLD_CRITICAL", "")
			t.Setenv("SCANIO_CONFIDENCE_THRESHOLD_HIGH", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			policy, enabled := parseRequiredPolicy(tt.flag)
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
			for k, v := range tt.wantThr {
				if policy.Thresholds[k] != v {
					t.Errorf("threshold[%q] = %v, want %v", k, policy.Thresholds[k], v)
				}
			}
		})
	}
}
