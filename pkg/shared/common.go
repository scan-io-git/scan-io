package shared

import (
	"bytes"
	"fmt"
	"log"
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

var ResultBuffer bytes.Buffer
var ResultBufferMutex sync.Mutex

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SCANIO",
	MagicCookieValue: "a65de33ff91e68ab6f5cd1fd5abb1235294816f5",
}

var PluginMap = map[string]plugin.Plugin{
	PluginTypeVCS:     &VCSPlugin{},
	PluginTypeScanner: &ScannerPlugin{},
}

type Versions struct {
	Version       string `json:"version"`
	GolangVersion string `json:"golang_version"`
	BuildTime     string `json:"build_time"`
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

	// TODO: use universal approach
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
