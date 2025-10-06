package ci

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// Resolution contains CI environment metadata resolved for VCS operations.
type Resolution struct {
	Kind        CIKind
	PluginName  string
	Domain      string
	Namespace   string
	Repository  string
	PullRequest string
	Hydrated    bool
}

// ResolveFromEnvironment determines the CI kind and collects metadata using the
// process environment. A non-empty providedPlugin is validated and preferred,
// while conflicts with the detected provider are logged. When neither a
// supported plugin is provided nor an environment can be detected, an error is
// returned so callers can prompt for explicit configuration.
func ResolveFromEnvironment(log hclog.Logger, providedPlugin string) (Resolution, error) {
	plugin := strings.TrimSpace(providedPlugin)
	result := Resolution{PluginName: plugin}

	providedKind := CIUnknown
	if plugin != "" {
		parsed, err := ParseCIKind(plugin)
		if err != nil {
			if log != nil {
				log.Warn("unable to interpret vcs option; falling back to CI detection", "vcs", plugin, "error", err)
			}
		} else {
			providedKind = parsed
			result.PluginName = parsed.String()
		}
	}

	detectedKind := DetectCIKind()
	result.Kind = detectedKind

	if plugin == "" {
		if detectedKind == CIUnknown {
			if log != nil {
				log.Error("unable to detect VCS plugin from CI environment; specify --vcs option")
			}
			return Resolution{}, fmt.Errorf("ci: unable to detect VCS plugin from CI environment; specify --vcs option")
		}
		result.PluginName = detectedKind.String()
		providedKind = detectedKind
		if log != nil {
			log.Info("detected VCS plugin from CI environment", "plugin", result.PluginName)
		}
	} else if providedKind != CIUnknown && detectedKind != CIUnknown && providedKind != detectedKind {
		if log != nil {
			log.Warn("provided VCS plugin differs from detected CI environment",
				"detected", detectedKind.String(), "provided", result.PluginName)
		}
	}

	hydrationKind := detectedKind
	if hydrationKind == CIUnknown {
		hydrationKind = providedKind
	}
	if hydrationKind == CIUnknown {
		return result, nil
	}

	env, err := GetCIDefaultEnvVars(hydrationKind)
	if err != nil {
		if log != nil {
			log.Debug("unable to hydrate from ci environment", "kind", hydrationKind.String(), "error", err)
		}
		return result, nil
	}

	result.Kind = env.Kind
	result.Hydrated = true

	if domain := hostFromEnvironment(env); domain != "" {
		result.Domain = domain
		if log != nil {
			log.Debug("hydrated domain from CI environment", "domain", domain)
		}
	}
	if env.Namespace != "" {
		result.Namespace = env.Namespace
		if log != nil {
			log.Debug("hydrated namespace from CI environment", "namespace", env.Namespace)
		}
	}
	if env.RepositoryName != "" {
		result.Repository = env.RepositoryName
		if log != nil {
			log.Debug("hydrated repository from CI environment", "repository", env.RepositoryName)
		}
	}
	if pr := derivePullRequestID(env.Kind, env); pr != "" {
		result.PullRequest = pr
		if log != nil {
			log.Debug("hydrated pull request id from CI environment", "pr", pr)
		}
	}

	return result, nil
}

func hostFromEnvironment(env CIEnvironment) string {
	sources := []string{env.VCSServerURL, env.RepositoryFullPath}
	for _, src := range sources {
		if strings.TrimSpace(src) == "" {
			continue
		}
		if parsed, err := url.Parse(src); err == nil && parsed.Host != "" {
			return parsed.Host
		}
	}
	return ""
}

func derivePullRequestID(kind CIKind, env CIEnvironment) string {
	switch kind {
	case CIGitHub, CIBitbucket:
		if pr := extractPRFromRef(env.Reference); pr != "" {
			return pr
		}
		if allDigits(env.ReferenceName) {
			return env.ReferenceName
		}
	case CIGitLab:
		if strings.HasPrefix(env.Reference, "refs/merge-requests/") && allDigits(env.ReferenceName) {
			return env.ReferenceName
		}
	}
	return ""
}

func extractPRFromRef(ref string) string {
	parts := strings.Split(ref, "/")
	for i := 0; i < len(parts); i++ {
		if parts[i] == "pull" || parts[i] == "merge-requests" {
			if i+1 < len(parts) && allDigits(parts[i+1]) {
				return parts[i+1]
			}
		}
	}
	return ""
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
