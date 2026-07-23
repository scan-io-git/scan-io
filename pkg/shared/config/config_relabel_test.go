package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Relabeled images relocate the home tree off /scanio and rely on
// SCANIO_CONFIG_PATH to find the moved config. Lock that contract.
func TestLoadConfig_SCANIO_CONFIG_PATH_Overrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("scanio:\n  home_folder: /myscanner\n"), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	t.Setenv("SCANIO_CONFIG_PATH", cfgPath)

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Scanio.HomeFolder != "/myscanner" {
		t.Fatalf("home_folder = %q, want /myscanner (SCANIO_CONFIG_PATH not honored)", cfg.Scanio.HomeFolder)
	}
}
