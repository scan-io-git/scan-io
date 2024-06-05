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
	PluginTypeVCS     string = "vcs"
	PluginTypeScanner string = "scanner"
)

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SCANIO",
	MagicCookieValue: "a65de33ff91e68ab6f5cd1fd5abb1235294816f5",
}

var PluginMap = map[string]plugin.Plugin{
	PluginTypeVCS:     &VCSPlugin{},
	PluginTypeScanner: &ScannerPlugin{},
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
		pluginsMeta["unknown"] = PluginMeta{Version: "unknown", PluginType: "unknown"}
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
		return PluginMeta{Version: "unknown", PluginType: "unknown"}
	}
	if err := json.Unmarshal(data, &pm); err != nil {
		return PluginMeta{Version: "unknown", PluginType: "unknown"}
	}
	return pm
}

func WithPlugin(cfg *config.Config, loggerName string, pluginType string, pluginName string, f func(interface{}) error) error {
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
		log.Fatal(err)
		return err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(pluginType)
	if err != nil {
		log.Fatal(err)
		return err
	}

	// TODO: Use universal approach
	var setupErr error
	switch pluginType {
	case "vcs":
		pluginInstance, ok := raw.(VCS)
		if !ok {
			err := fmt.Errorf("plugin does not implement VCS interface")
			logger.Error(err.Error())
			return err
		}
		_, setupErr = pluginInstance.Setup(*cfg)
	case "scanner":
		pluginInstance, ok := raw.(Scanner)
		if !ok {
			err := fmt.Errorf("plugin does not implement Scanner interface")
			logger.Error(err.Error())
			return err
		}
		_, setupErr = pluginInstance.Setup(*cfg)

	default:
		return fmt.Errorf("unsupported plugin type: %s", pluginType)
	}

	if setupErr != nil {
		logger.Error("failed to setup plugin", "error", setupErr)
		return setupErr
	}

	err = f(raw)
	if err != nil {
		return err
	}

	return nil
}

func ForEveryStringWithBoundedGoroutines(limit int, values []interface{}, f func(i int, value interface{})) {
	guard := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i, value := range values {
		guard <- struct{}{} // would block if guard channel is already filled
		wg.Add(1)
		go func(i int, value interface{}) {
			defer wg.Done()
			f(i, value)
			<-guard
		}(i, value)
	}
	wg.Wait()
}
