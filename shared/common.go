package shared

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/libs/vcs"
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
	PluginTypeVCS:     &vcs.VCSPlugin{},
	PluginTypeScanner: &ScannerPlugin{},
}

func NewLogger(name string) hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Name:   name,
		Output: os.Stdout,
		Level:  hclog.Debug,
	})
}

func WithPlugin(loggerName string, pluginType string, pluginName string, f func(interface{})) {
	logger := NewLogger(loggerName)

	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	pluginsFolder := filepath.Join(home, "/.scanio/plugins")

	pluginPath := filepath.Join(pluginsFolder, pluginName)
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
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(pluginType)
	if err != nil {
		log.Fatal(err)
	}

	f(raw)
}

func ForEveryStringWithBoundedGoroutines(limit int, values []string, f func(i int, value string)) {
	guard := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i, value := range values {
		guard <- struct{}{} // would block if guard channel is already filled
		wg.Add(1)
		go func(i int, value string) {
			defer wg.Done()
			f(i, value)
			<-guard
		}(i, value)
	}
	wg.Wait()
}

func GetProjectsHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	projectsFolder := filepath.Join(home, "/.scanio/projects")
	if _, err := os.Stat(projectsFolder); os.IsNotExist(err) {
		NewLogger("core").Info("projectsFolder does not exists. Creating...", "projectsFolder", projectsFolder)
		if err := os.MkdirAll(projectsFolder, os.ModePerm); err != nil {
			panic(err)
		}
	}
	return projectsFolder
}

func GetPluginsHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
	}
	pluginsFolder := filepath.Join(home, "/.scanio/plugins")
	if _, err := os.Stat(pluginsFolder); os.IsNotExist(err) {
		NewLogger("core").Info("pluginsFolder does not exists. Creating...", "pluginsFolder", pluginsFolder)
		if err := os.MkdirAll(pluginsFolder, os.ModePerm); err != nil {
			panic(err)
		}
	}
	return pluginsFolder
}

func GetRepoPath(VCSURL, repoWithNamespace string) string {
	return filepath.Join(GetProjectsHome(), VCSURL, repoWithNamespace)
}

func ReadFileLines(inputFile string) ([]string, error) {
	readFile, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer readFile.Close()

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	lines := []string{}
	for fileScanner.Scan() {
		lines = append(lines, fileScanner.Text())
	}

	return lines, nil
}
