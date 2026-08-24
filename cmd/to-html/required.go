package tohtml

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	scaniosarif "github.com/scan-io-git/scan-io/internal/sarif"
)

// validSeverities are the only severity buckets a finding can ever carry (see
// internal/sarif EnrichResultsLevelProperty). A --required/env severity outside
// this set can never match a finding, so it is rejected rather than accepted as
// a silent no-op.
var validSeverities = map[string]bool{
	"critical": true,
	"high":     true,
	"medium":   true,
	"low":      true,
	"info":     true,
}

// parseThreshold parses and range-checks a confidence threshold string. Valid
// thresholds are in [0.0, 1.0]; anything else is an error, since an
// out-of-range threshold can never be crossed (or is always crossed) and is as
// meaningless as an unparseable one.
func parseThreshold(raw string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, fmt.Errorf("not a number")
	}
	if f < 0.0 || f > 1.0 {
		return 0, fmt.Errorf("must be between 0.0 and 1.0")
	}
	return f, nil
}

// parseRequiredPolicy builds a classification policy from the --required flag,
// falling back to env vars when the flag is empty. Returns (policy, false, nil)
// when the feature is not configured (empty flag and no env vars) -- that is a
// legitimate "off", not an error. Returns a non-nil error when the flag or env
// vars are set but malformed: an unparseable or out-of-range confidence
// threshold, or a severity name that is not one of critical/high/medium/low/info.
// Flag wins over env (repo-wide precedence rule).
//
// A severity listed without a threshold (e.g. "high") marks all findings of that
// severity as Required regardless of their confidence score. A per-severity
// threshold is only applied when explicitly supplied via the "sev:N" syntax or the
// SCANIO_CONFIDENCE_THRESHOLD_<SEV> env var; no defaults are injected.
//
// Flag format: "sev[:threshold],..." e.g. "critical,high" or "critical:0.50,high:0.90".
// Env: SCANIO_BLOCKER_SEVERITIES="critical,high",
//
//	SCANIO_CONFIDENCE_THRESHOLD_<SEV>="0.95".
func parseRequiredPolicy(flagValue string) (scaniosarif.RequiredPolicy, bool, error) {
	thresholds := map[string]float64{}
	blockers := map[string]bool{}

	if flag := strings.TrimSpace(flagValue); flag != "" {
		for _, item := range strings.Split(flag, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			sev, thr, hasThr := strings.Cut(item, ":")
			sev = strings.ToLower(strings.TrimSpace(sev))
			if sev == "" {
				continue
			}
			if !validSeverities[sev] {
				return scaniosarif.RequiredPolicy{}, false, fmt.Errorf("invalid severity %q in --required: must be one of critical, high, medium, low, info", sev)
			}
			blockers[sev] = true
			if hasThr {
				thr = strings.TrimSpace(thr)
				f, err := parseThreshold(thr)
				if err != nil {
					return scaniosarif.RequiredPolicy{}, false, fmt.Errorf("invalid confidence threshold %q for severity %q: %v", thr, sev, err)
				}
				thresholds[sev] = f
			}
		}
		if len(blockers) == 0 {
			return scaniosarif.RequiredPolicy{}, false, nil
		}
		return scaniosarif.RequiredPolicy{BlockerSeverities: blockers, Thresholds: thresholds}, true, nil
	}

	// Env fallback.
	envSevs := strings.TrimSpace(os.Getenv("SCANIO_BLOCKER_SEVERITIES"))
	if envSevs == "" {
		return scaniosarif.RequiredPolicy{}, false, nil
	}
	for _, sev := range strings.Split(envSevs, ",") {
		sev = strings.ToLower(strings.TrimSpace(sev))
		if sev == "" {
			continue
		}
		if !validSeverities[sev] {
			return scaniosarif.RequiredPolicy{}, false, fmt.Errorf("invalid severity %q in SCANIO_BLOCKER_SEVERITIES: must be one of critical, high, medium, low, info", sev)
		}
		blockers[sev] = true
	}
	if len(blockers) == 0 {
		return scaniosarif.RequiredPolicy{}, false, nil
	}
	for _, sev := range []string{"critical", "high", "medium", "low", "info"} {
		envVar := "SCANIO_CONFIDENCE_THRESHOLD_" + strings.ToUpper(sev)
		if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
			f, err := parseThreshold(v)
			if err != nil {
				return scaniosarif.RequiredPolicy{}, false, fmt.Errorf("invalid confidence threshold %q for %s: %v", v, envVar, err)
			}
			thresholds[sev] = f
		}
	}
	return scaniosarif.RequiredPolicy{BlockerSeverities: blockers, Thresholds: thresholds}, true, nil
}
