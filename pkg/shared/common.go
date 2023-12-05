package shared

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
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

func NewLogger(name string) hclog.Logger {
	loglevel := hclog.Info
	jsonFormat := false
	if os.Getenv("SCANIO_LOGLEVEL") == "DEBUG" {
		loglevel = hclog.Debug
		jsonFormat = false
	}
	return hclog.New(&hclog.LoggerOptions{
		Name:        name,
		DisableTime: true,
		Output:      os.Stdout,
		Level:       loglevel,
		JSONFormat:  jsonFormat,
	})
}

func GetScanioHome() string {
	envScanioHome := os.Getenv("SCANIO_HOME")
	if envScanioHome != "" {
		return envScanioHome
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	defaultScanioHome := filepath.Join(home, "/.scanio")
	return defaultScanioHome
}

func getScanioPluginsFolder() string {
	envScanioPlugins := os.Getenv("SCANIO_PLUGINS_FOLDER")
	if envScanioPlugins != "" {
		return envScanioPlugins
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	defaultScanioPlugins := filepath.Join(home, "/.scanio/plugins")
	return defaultScanioPlugins
}

func WithPlugin(loggerName string, pluginType string, pluginName string, f func(interface{}) error) error {
	logger := NewLogger(loggerName)

	pluginPath := filepath.Join(getScanioPluginsFolder(), pluginName)
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

	if err := f(raw); err != nil {
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

func GetProjectsHome() string {
	projectsFolder := filepath.Join(GetScanioHome(), "/projects")
	if _, err := os.Stat(projectsFolder); os.IsNotExist(err) {
		NewLogger("core").Info("projectsFolder does not exists. Creating...", "projectsFolder", projectsFolder)
		if err := os.MkdirAll(projectsFolder, os.ModePerm); err != nil {
			panic(err)
		}
	}
	return projectsFolder
}

func GetResultsHome() string {
	resultsFolder := filepath.Join(GetScanioHome(), "/results")
	if _, err := os.Stat(resultsFolder); os.IsNotExist(err) {
		NewLogger("core").Info("resultsFolder does not exists. Creating...", "resultsFolder", resultsFolder)
		if err := os.MkdirAll(resultsFolder, os.ModePerm); err != nil {
			panic(err)
		}
	}
	return resultsFolder
}

func GetRepoPath(VCSURL, repoWithNamespace string) string {
	return filepath.Join(GetProjectsHome(), VCSURL, repoWithNamespace)
}
