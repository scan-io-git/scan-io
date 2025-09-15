package artifacts

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

// GetArtifactName build returns artifact name.
// Example: analyse_semgrep_2025-09-15T08:28:46Z.scanio-artifact.
func GetArtifactName(command, plugin string, t time.Time) string {
	ts := t.UTC().Format(time.RFC3339)
	metaDataFileName := fmt.Sprintf("%s_%s_%s.scanio-artifact", command, plugin, ts)
	return metaDataFileName
}

// SaveArtifactJSON writes the provided result to a <artifacts>/<base>.json.
// Returns full path.
func SaveArtifactJSON(cfg *config.Config, logger hclog.Logger, command, plugin string, result shared.GenericLaunchesResult) (string, error) {
	dir := config.GetScanioArtifactsHome(cfg)
	base := GetArtifactName(command, plugin, time.Now())
	path := filepath.Join(dir, base+".json")

	resultData, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return path, fmt.Errorf("error marshaling the result data: %w", err)
	}

	if err := files.WriteJsonFile(path, resultData); err != nil {
		return path, fmt.Errorf("error writing result to log file: %w", err)
	}
	logger.Info("artifact saved to file", "path", path)

	return path, nil
}
