package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gitsight/go-vcsurl"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/scan-io-git/scan-io/shared"
)

// Here is a real implementation of Scanner
type ScannerSemgrep struct {
	logger hclog.Logger
}

func (g *ScannerSemgrep) Scan(project string) bool {

	home, err := os.UserHomeDir()
	if err != nil {
		panic("unable to get home folder")
		// return false
	}
	projectsFolder := filepath.Join(home, "/.scanio/projects")
	if _, err := os.Stat(projectsFolder); os.IsNotExist(err) {
		g.logger.Info("projectsFolder '%s' does not exists. Creating...", projectsFolder)
		if err := os.MkdirAll(projectsFolder, os.ModePerm); err != nil {
			panic(err)
			// return false
		}
	}
	resultsFolder := filepath.Join(home, "/.scanio/results")
	if _, err := os.Stat(resultsFolder); os.IsNotExist(err) {
		g.logger.Info("resultsFolder does not exists. Creating...", "resultsFolder", resultsFolder)
		if err := os.MkdirAll(resultsFolder, os.ModePerm); err != nil {
			panic(err)
			// return false
		}
	}

	info, err := vcsurl.Parse(project)
	if err != nil {
		g.logger.Error("unable to parse project '%s'", project)
		panic(err)
		// return false
	}

	projectFolder := filepath.Join(projectsFolder, info.ID)
	resultsFile := filepath.Join(resultsFolder, info.ID, fmt.Sprintf("semgrep.sarif"))

	cmd := exec.Command("semgrep", "--config", "auto", "-o", resultsFile, "--sarif", projectFolder)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err = cmd.Run()
	g.logger.Info("Scan finished", "project", projectFolder, "resultsFile", resultsFile)

	if err != nil {
		g.logger.Error("semgrep execution error", "err", err, "projectFolder", projectFolder)
		return false
	}

	return true
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
// var handshakeConfig = plugin.HandshakeConfig{
// 	ProtocolVersion:  1,
// 	MagicCookieKey:   "BASIC_PLUGIN",
// 	MagicCookieValue: "hello",
// }

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	Scanner := &ScannerSemgrep{
		logger: logger,
	}
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		"scanner": &shared.ScannerPlugin{Impl: Scanner},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         pluginMap,
	})
}
