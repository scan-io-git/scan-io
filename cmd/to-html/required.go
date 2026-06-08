package tohtml

import (
	"os"
	"strconv"
	"strings"

	scaniosarif "github.com/scan-io-git/scan-io/internal/sarif"
)

// parseRequiredPolicy builds a classification policy from the --required flag,
// falling back to env vars when the flag is empty. Returns (policy, false) when
// the feature is not configured. Flag wins over env (repo-wide precedence rule).
//
// Flag format: "sev[:threshold],..." e.g. "critical:0.50,high".
// Env: SCANIO_BLOCKER_SEVERITIES="critical,high",
//
//	SCANIO_CONFIDENCE_THRESHOLD_<SEV>="0.95".
func parseRequiredPolicy(flagValue string) (scaniosarif.RequiredPolicy, bool) {
	thresholds := scaniosarif.DefaultConfidenceThresholds()
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
			blockers[sev] = true
			if hasThr {
				if f, err := strconv.ParseFloat(strings.TrimSpace(thr), 64); err == nil {
					thresholds[sev] = f
				}
			}
		}
		if len(blockers) == 0 {
			return scaniosarif.RequiredPolicy{}, false
		}
		return scaniosarif.RequiredPolicy{BlockerSeverities: blockers, Thresholds: thresholds}, true
	}

	// Env fallback.
	envSevs := strings.TrimSpace(os.Getenv("SCANIO_BLOCKER_SEVERITIES"))
	if envSevs == "" {
		return scaniosarif.RequiredPolicy{}, false
	}
	for _, sev := range strings.Split(envSevs, ",") {
		sev = strings.ToLower(strings.TrimSpace(sev))
		if sev == "" {
			continue
		}
		blockers[sev] = true
	}
	if len(blockers) == 0 {
		return scaniosarif.RequiredPolicy{}, false
	}
	for _, sev := range []string{"critical", "high", "medium", "low", "info"} {
		if v := strings.TrimSpace(os.Getenv("SCANIO_CONFIDENCE_THRESHOLD_" + strings.ToUpper(sev))); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				thresholds[sev] = f
			}
		}
	}
	return scaniosarif.RequiredPolicy{BlockerSeverities: blockers, Thresholds: thresholds}, true
}
