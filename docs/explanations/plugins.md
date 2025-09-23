# Plugins

We can think about Scanio as about 2 major components:
1. Scanio Core
2. Scanio Plugins

The Scanio is built with a plugin system in mind. All companies have a unique setup, use different tools. Plugin system helps to unify the experience of using scanio, make it easy to switch between different plugins, and give a power of extensibility by impleneting new plugins. The main function of plugins is to glue core with different systems and hide the complexities of individual systems behind a single interface.
The Scanio Core is focused on implementing more of common code, command handling, input validation, logging, different utility functionalities. For integrations with other systems it invokes dedicated plugins.

Currently there are 2 main groups of plugins incorporated into scanio:
1. **Scanner Plugins**: These plugins integrate with security tools to perform scans. Examples include:
   - **[Bandit Plugin](docs/reference/plugin-bandit.md)**: Integrates with the Bandit scanner for analyzing Python code.
   - **[Semgrep Plugin](docs/reference/plugin-semgrep.md)**: Enables static analysis using Semgrep.
   - **[Trufflehog3 Plugin](docs/reference/plugin-trufflehog3.md)**: Scans for secrets in codebases.
2. **VCS Plugins**: These plugins interact with version control systems (VCS) to list repositories, clone code, or manage pull requests. Examples include:
   - **[GitLab Plugin](docs/reference/plugin-gitlab.md)**: Works with GitLab repositories.
   - **[GitHub Plugin](docs/reference/plugin-github.md)**: Integrates with GitHub for repository management.
   - **[Bitbucket Plugin](docs/reference/plugin-bitbucket.md)**: Facilitates interactions with Bitbucket.

You can read more about each individual plugins on their dedicated reference doc pages.

## Implementation
The plugins system is built with the help of `hashicorp/go-plugin` module. It manages the trigger of plugins execution, connection and function execution over RCP, and other. On scanio side connection interfaces were implemented.  
Interface for VCS plugins can be found in [pkg/shared/ivcs.go](pkg/shared/ivcs.go) defines the interface of plugins. Each plugin must define all of the function described in the `type VCS interface`. Interface file also defines all parameter types, needed for plugins function call.  
Same as for vcs plugins, there is a file that defines interface and related types for scanners in [iscanner.go](/pkg/shared/iscanner.go).
There is no plugins registration mechanism. Plugins are rather invoked by known filesystem path to the plugin, and a handshake mechanism.


### Plugin implementation notes
The plugin side of Scanio is where the actual functionality for individual scanners is implemented.

To start serving as a plugin, the `plugin.Serve` method should be called. It accepts several parameters: a shared handshake config, a logger, and a plugin defined by the plugin itself.
```go
plugin.Serve(&plugin.ServeConfig{
  HandshakeConfig: shared.HandshakeConfig,
  Plugins: map[string]plugin.Plugin{
    shared.PluginTypeScanner: &shared.ScannerPlugin{Impl: scannerInstance},
  },
  Logger: logger,
})
```

The `scannerInstance` structure must implement all method defined by the plugin interface definition. For scanner plugin it's `Setup` and `Scan` methods. A plugin may also contain other methods.
```go
type Scanner interface {
	Setup(configData config.Config) (bool, error)
	Scan(args ScannerScanRequest) (ScannerScanResponse, error)
}
```

The main purpose of `Setup` function is to pass the config file data, that can be later used by the plugin to handle main functionality.
