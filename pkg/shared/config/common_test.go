package config

import "testing"

func TestGetters_NilConfig_ReturnEmpty(t *testing.T) {
	if got := GetScanioHome(nil); got != "" {
		t.Errorf("GetScanioHome(nil) = %q, want \"\"", got)
	}
	if got := GetScanioPluginsHome(nil); got != "" {
		t.Errorf("GetScanioPluginsHome(nil) = %q, want \"\"", got)
	}
	if got := GetScanioProjectsHome(nil); got != "" {
		t.Errorf("GetScanioProjectsHome(nil) = %q, want \"\"", got)
	}
	if got := GetScanioResultsHome(nil); got != "" {
		t.Errorf("GetScanioResultsHome(nil) = %q, want \"\"", got)
	}
	if got := GetScanioTempHome(nil); got != "" {
		t.Errorf("GetScanioTempHome(nil) = %q, want \"\"", got)
	}
	if got := GetScanioArtifactsHome(nil); got != "" {
		t.Errorf("GetScanioArtifactsHome(nil) = %q, want \"\"", got)
	}
	if got := GetScanioMode(nil); got != "" {
		t.Errorf("GetScanioMode(nil) = %q, want \"\"", got)
	}
	if got := GetRepositoryPath(nil, "git.example", "ns/repo"); got != "" {
		t.Errorf("GetRepositoryPath(nil, ...) = %q, want \"\"", got)
	}
	if got := GetPRTempPath(nil, "git.example", "ns", "repo", 1); got != "" {
		t.Errorf("GetPRTempPath(nil, ...) = %q, want \"\"", got)
	}
}

func TestIsCI_NilConfig_ReturnsFalse(t *testing.T) {
	if IsCI(nil) {
		t.Error("IsCI(nil) = true, want false")
	}
}

func TestIsCI_NonCIMode_ReturnsFalse(t *testing.T) {
	cfg := &Config{}
	cfg.Scanio.Mode = "user"
	if IsCI(cfg) {
		t.Error("IsCI(user) = true, want false")
	}
}

func TestIsCI_CIMode_ReturnsTrue(t *testing.T) {
	cfg := &Config{}
	cfg.Scanio.Mode = "CI"
	if !IsCI(cfg) {
		t.Error("IsCI(CI) = false, want true")
	}
}
