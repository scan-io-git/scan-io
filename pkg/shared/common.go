package shared

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-plugin"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

const (
	PluginTypeVCS     = "vcs"
	PluginTypeScanner = "scanner"
	unknownVersion    = "unknown"
)

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SCANIO",
	MagicCookieValue: "a65de33ff91e68ab6f5cd1fd5abb1235294816f5",
}

// PluginMap defines the available plugins.
var PluginMap = map[string]plugin.Plugin{
	PluginTypeVCS:     &VCSPlugin{},
	PluginTypeScanner: &ScannerPlugin{},
}

// GenericResult represents the result of a generic operation.
type GenericResult struct {
	Args    interface{} `json:"args"`
	Result  interface{} `json:"result"`
	Status  string      `json:"status"`
	Message string      `json:"message"`
}

// GenericLaunchesResult represents a list of launches.
type GenericLaunchesResult struct {
	Launches []GenericResult `json:"launches"`
}

// Versions holds meta information for binaries.
type Versions struct {
	Version       string `json:"version"`
	GolangVersion string `json:"golang_version"`
	BuildTime     string `json:"build_time"`
}

// PluginMeta holds version information for a plugin.
type PluginMeta struct {
	Version    string `json:"version"`
	PluginType string `json:"plugin_type"`
}

// GetPluginVersions iterates through the plugin directories and reads their version files.
func GetPluginVersions(pluginsDir, pluginType string) map[string]PluginMeta {
	pluginsMeta := make(map[string]PluginMeta)
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		log.Printf("Failed to read plugins directory: %v", err)
		pluginsMeta[unknownVersion] = PluginMeta{Version: unknownVersion, PluginType: unknownVersion}
		return pluginsMeta
	}
	for _, entry := range entries {
		if entry.IsDir() {
			pluginName := entry.Name()
			versionFilePath := filepath.Join(pluginsDir, pluginName, "VERSION")
			version := readVersionFile(versionFilePath)
			if pluginType == "" || version.PluginType == pluginType {
				pluginsMeta[pluginName] = version
			}
		}
	}
	return pluginsMeta
}

// readVersionFile reads and parses the version file as JSON.
func readVersionFile(versionFilePath string) PluginMeta {
	var pm PluginMeta
	data, err := os.ReadFile(versionFilePath)
	if err != nil {
		return PluginMeta{Version: unknownVersion, PluginType: unknownVersion}
	}
	if err := json.Unmarshal(data, &pm); err != nil {
		return PluginMeta{Version: unknownVersion, PluginType: unknownVersion}
	}
	return pm
}

// WithPlugin initializes the plugin client, sets up the plugin, and executes the provided function.
func WithPlugin(cfg *config.Config, loggerName, pluginType, pluginName string, f func(interface{}) error) error {
	logger := logger.NewLogger(cfg, loggerName)

	pluginPath := filepath.Join(config.GetScanioPluginsHome(cfg), pluginName, pluginName)
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         PluginMap,
		Cmd:             exec.Command(pluginPath),
		Logger:          logger,
	})
	defer client.Kill()

	rpcClient, err := client.Client()
	if err != nil {
		logger.Error("failed to get RPC client", "error", err)
		return fmt.Errorf("failed to get RPC client: %w", err)
	}

	raw, err := rpcClient.Dispense(pluginType)
	if err != nil {
		logger.Error("failed to dispense plugin", "pluginType", pluginType, "error", err)
		return fmt.Errorf("failed to dispense plugin: %w", err)
	}

	if err = setupPlugin(cfg, pluginType, raw); err != nil {
		logger.Error("failed to setup plugin", "pluginType", pluginType, "error", err)
		return fmt.Errorf("failed to setup plugin: %w", err)
	}

	return f(raw)
}

// setupPlugin sets up the plugin based on its type.
func setupPlugin(cfg *config.Config, pluginType string, raw interface{}) error {
	var err error
	switch pluginType {
	case PluginTypeVCS:
		pluginInstance, ok := raw.(VCS)
		if !ok {
			return fmt.Errorf("plugin does not implement VCS interface")
		}
		_, err = pluginInstance.Setup(*cfg)
	case PluginTypeScanner:
		pluginInstance, ok := raw.(Scanner)
		if !ok {
			return fmt.Errorf("plugin does not implement Scanner interface")
		}
		_, err = pluginInstance.Setup(*cfg)
	default:
		return fmt.Errorf("unsupported plugin type: %s", pluginType)
	}
	return err
}

// ForEveryStringWithBoundedGoroutines limits the number of concurrent goroutines and executes the provided function.
func ForEveryStringWithBoundedGoroutines(limit int, values []interface{}, f func(i int, value interface{})) {
	guard := make(chan struct{}, limit)
	var wg sync.WaitGroup

	for i, value := range values {
		guard <- struct{}{}
		wg.Add(1)
		go func(i int, value interface{}) {
			defer wg.Done()
			// defer func() { <-guard }()
			f(i, value)
			<-guard
		}(i, value)
	}
	wg.Wait()
}
