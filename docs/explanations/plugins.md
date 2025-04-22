# Plugins

We can think about Scanio as about 2 major components:
1. Scanio Core
2. Scanio Plugins

The Scanio is built with a plugin system in mind. All companies have a unique setup, use different tools. Plugin system helps to unify the experience of using scanio, make it easy to switch between different plugins, and give a power of extensibility by impleneting new plugins. The main function of plugins is to glue core with different systems and hide the complexities of individual systems behind a single interface.
The Scanio Core is focused on implementing more of common code, command handling, input validation, logging, different utility functionalities. For integrations with other systems it invokes dedicated plugins.

Currently there are 2 main groups of plugins incorporated into scanio:
- Scanner plugins
- VCS plugins

You can read more about each individual plugins on their dedicated reference doc pages.

## Implementation
The plugins system is built with the help of `hashicorp/go-plugin` module. It manages the trigger of plugins execution, connection and function execution over RCP, and other. On scanio side connection interfaces were implemented.  
Interface for VCS plugins can be found in [pkg/shared/ivcs.go](pkg/shared/ivcs.go) defines the interface of plugins. Each plugin must define all of the function described in the `type VCS interface`. Interface file also defines all parameter types, needed for plugins function call.  
Same as for vcs plugins, there is a file that defines interface and related types for scanners in [iscanner.go](/pkg/shared/iscanner.go).


### Method definition on scanio core side
TBD

### Method definition on plugin side
TBD
