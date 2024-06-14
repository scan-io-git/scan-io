package version

import (
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
	Versions       shared.Versions   `json:"versions"`
	PluginVersions map[string]string `json:"plugin_versions"`
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
				Versions:       versionInfo,
				PluginVersions: getPluginVersions(config.GetScanioPluginsHome(AppConfig)),
			}

			printVersionInfo(&version)
		},
	}
}

// getVersion reads the version from the given version file.
func readVersionFile(versionFilePath string) string {
	data, err := os.ReadFile(versionFilePath)
	if err != nil {
		return "unknown"
	}
	return string(data)
}

// getPluginVersions iterates through the plugin directories and reads their version files.
func getPluginVersions(pluginsDir string) map[string]string {
	pluginVersions := make(map[string]string)
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		log.Printf("Failed to read plugins directory: %v", err)
		pluginVersions["unknown"] = "unknown"
		return pluginVersions
	}
	for _, entry := range entries {
		if entry.IsDir() {
			pluginName := entry.Name()
			versionFilePath := filepath.Join(pluginsDir, pluginName, "VERSION")
			version := readVersionFile(versionFilePath)
			pluginVersions[pluginName] = version
		}
	}
	return pluginVersions
}

// printVersionInfo prints the version information for the core application and plugins.
func printVersionInfo(versions *CoreVersions) {
	fmt.Printf("Core Version: v%s\n", versions.Versions.Version)
	fmt.Println("Plugin Versions:")
	for plugin, version := range versions.PluginVersions {
		fmt.Printf("  %s: v%s\n", plugin, version)
	}
	fmt.Printf("Go Version: %s\n", versions.Versions.GolangVersion)
	fmt.Printf("Build Time: %s\n", versions.Versions.BuildTime)
}
