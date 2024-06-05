package version

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

var (
	AppConfig     *config.Config
	CoreVersion   = "unknown"
	GolangVersion = "unknown"
	BuildTime     = "unknown"
)

// Versions struct holds version information for the core application and plugins.
type CoreVersions struct {
	Versions    shared.Versions       `json:"versions"`
	PluginsMeta map[string]PluginMeta `json:"plugins_meta"`
}

// PluginVersion holds version information for a plugin.
type PluginMeta struct {
	Version    string `json:"version"`
	PluginType string `json:"plugin_type"`
}

// Init initializes the global configuration variable.
func Init(cfg *config.Config) {
	AppConfig = cfg
}

// NewVersionCmd creates a new cobra.Command for the version command.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "version",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Short:                 "Print the version number of the application and plugins",
		Run: func(cmd *cobra.Command, args []string) {
			versionInfo := shared.Versions{
				Version:       CoreVersion,
				GolangVersion: GolangVersion,
				BuildTime:     BuildTime,
			}
			version := CoreVersions{
				Versions:    versionInfo,
				PluginsMeta: getPluginVersions(config.GetScanioPluginsHome(AppConfig)),
			}

			printVersionInfo(&version)
		},
	}
}

// readVersionFile reads and parses the version file as JSON.
func readVersionFile(versionFilePath string) PluginMeta {
	var pm PluginMeta
	data, err := os.ReadFile(versionFilePath)
	if err != nil {
		return PluginMeta{Version: "unknown", PluginType: "unknown"}
	}
	if err := json.Unmarshal(data, &pm); err != nil {
		return PluginMeta{Version: "unknown", PluginType: "unknown"}
	}
	return pm
}

// getPluginVersions iterates through the plugin directories and reads their version files.
func getPluginVersions(pluginsDir string) map[string]PluginMeta {
	pluginsMeta := make(map[string]PluginMeta)
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		log.Printf("Failed to read plugins directory: %v", err)
		pluginsMeta["unknown"] = PluginMeta{Version: "unknown", PluginType: "unknown"}
		return pluginsMeta
	}
	for _, entry := range entries {
		if entry.IsDir() {
			pluginName := entry.Name()
			versionFilePath := filepath.Join(pluginsDir, pluginName, "VERSION")
			version := readVersionFile(versionFilePath)
			pluginsMeta[pluginName] = version
		}
	}
	return pluginsMeta
}

// printVersionInfo prints the version information for the core application and plugins.
func printVersionInfo(versions *CoreVersions) {
	fmt.Printf("Core Version: v%s\n", versions.Versions.Version)
	fmt.Println("Plugin Versions:")
	for plugin, version := range versions.PluginsMeta {
		fmt.Printf("  %s: v%s (Type: %s)\n", plugin, version.Version, version.PluginType)
	}
	fmt.Printf("Go Version: %s\n", versions.Versions.GolangVersion)
	fmt.Printf("Build Time: %s\n", versions.Versions.BuildTime)
}
