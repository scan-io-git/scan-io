package version

import (
	"fmt"

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
	Versions      shared.Versions              `json:"versions"`
	PluginDetails map[string]shared.PluginMeta `json:"plugin_details"`
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
				Versions:      versionInfo,
				PluginDetails: shared.GetPluginVersions(config.GetScanioPluginsHome(AppConfig), ""),
			}

			printVersionInfo(&version)
		},
	}
}

// printVersionInfo prints the version information for the core application and plugins.
func printVersionInfo(versions *CoreVersions) {
	fmt.Printf("Core Version: v%s\n", versions.Versions.Version)
	fmt.Println("Plugin Versions:")
	for plugin, version := range versions.PluginDetails {
		fmt.Printf("  %s: v%s (Type: %s)\n", plugin, version.Version, version.PluginType)
	}
	fmt.Printf("Go Version: %s\n", versions.Versions.GolangVersion)
	fmt.Printf("Build Time: %s\n", versions.Versions.BuildTime)
}
