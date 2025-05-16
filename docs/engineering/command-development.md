# Writing a New Command in Scan-io

## 1. Command Structure

Each command should be organized in its own package under the `cmd` directory. The basic structure should include:

```
cmd/
  your-command/
    your-command.go  # Main command implementation
    validation.go    # Input validation logic
    utils.go         # Helper functions
```

## 2. Command Implementation Steps

### 2.1. Define Command Options

Create a struct to hold command options, like this:

```go
type YourCommandOptions struct {
    // Required options
    PluginName string `json:"plugin_name,omitempty"`
    OutputPath string `json:"output_path,omitempty"`
    // Add other options as needed
}
```
 
For reference, review how the fetch command defines its options in [cmd/fetch/fetch.go](https://github.com/scan-io-git/scan-io/blob/v0.3.0/cmd/fetch/fetch.go#L18-L27).

### 2.2. Create Cobra Command

Define the command using cobra:

```go
var YourCommand = &cobra.Command{
    SilenceUsage:          true,
    DisableFlagsInUseLine: true,  // set "true" and define "Usage" when default Usage is not good enough, and you want to have a full control of Usage
    Use:                   "your-command --plugin/-p PLUGIN_NAME [options] {--input-file/-i PATH | PATH}",  // add comprehensive usage example to reflect all arguments
    Example:               exampleYourCommandUsage, // add some usage examples
    Short:                 "Brief description of your command",
    RunE:                  runYourCommand,  // command logic implementation will be discussed in the following steps.
}
```

For reference, review how the fetch command defines the cobra command in [cmd/fetch/fetch.go](https://github.com/scan-io-git/scan-io/blob/v0.3.0/cmd/fetch/fetch.go#L61-L68).

### 2.3. Initialize Command

Create an Init function to set up the command:

```go
func Init(cfg *config.Config) {
    AppConfig = cfg  // store the AppConfig inside the module
    YourCommand.Long = generateLongDescription(AppConfig)  // Use this, when you want to generate a dynamic description, based on the environment, for example available plugins. 
}
```

For reference, review how the fetch command defines the Init method in [cmd/fetch/fetch.go](https://github.com/scan-io-git/scan-io/blob/v0.3.0/cmd/fetch/fetch.go#L71-L74).

The Init method should be called from the root command's initialization in `cmd/root.go`. Here's how to call it:

```go
// In cmd/root.go
import (
	"github.com/scan-io-git/scan-io/cmd/yourcommand"
)

func initConfig() {
    // ... other initialization code ...
    
    // Initialize your command with the app config
    yourcommand.Init(AppConfig)
}

func init() {
    // ... other initialization code ...
    
    // Add the command to the root command
    rootCmd.AddCommand(yourcommand.YourCommand)
}
```

### 2.4. Implement Command Logic

Create the main command execution function:

```go
func runYourCommand(cmd *cobra.Command, args []string) error {
    // 1. Check for help request
    // - It detects if the user has provided any command-line flags
    // - If no flags are provided and no arguments are given, it shows the help message
    if len(args) == 0 && !shared.HasFlags(cmd.Flags()) {
        return cmd.Help()
    }

    // 2. Initialize logger
    logger := logger.NewLogger(AppConfig, "core-your-command")

    // 3. Validate arguments
    // validation function should be implemented in validation.go file
    if err := validateYourCommandArgs(&yourCommandOptions, args); err != nil {
        logger.Error("invalid arguments", "error", err)
        return errors.NewCommandError(yourCommandOptions, nil, fmt.Errorf("invalid arguments: %w", err), 1)
    }

    // 4. Determine operation mode if needed
    // The mode determines how the command should process its input:
    // - ModeSingleURL: when a URL is provided as an argument
    // - ModeInputFile: when using an input file specified by flags
    mode := determineMode(args)

    // 5. Implement the business logic of the command
    // Try to existing modules like scanner, fetcher, and so on
    // Or implement the logic in a separate reusable module.

    // ...
    
    // 6. Handle results
    metaDataFileName := fmt.Sprintf("YOUR_COMMAND_%s", strings.ToUpper(handler.PluginName))
    if config.IsCI(AppConfig) {
        startTime := time.Now().UTC().Format(time.RFC3339)
        metaDataFileName = fmt.Sprintf("YOUR_COMMAND_%s_%v", strings.ToUpper(handler.PluginName), startTime)
    }

    if err := shared.WriteGenericResult(AppConfig, logger, result, metaDataFileName); err != nil {
        logger.Error("failed to write result", "error", err)
    }

    if err != nil {
        logger.Error("command failed", "error", err)
        return errors.NewCommandErrorWithResult(result, fmt.Errorf("command failed: %w", err), 2)
    }

    // 7. Log success and handle CI output
    logger.Info("command completed successfully")
    logger.Debug("command result", "result", result)
    if config.IsCI(AppConfig) {
        shared.PrintResultAsJSON(logger, result)
    }
    return nil
}
```

### 2.5. Add Command Flags

Initialize command flags in the init function:

```go
func init() {
    YourCommand.Flags().StringVarP(&yourCommandOptions.PluginName, "plugin", "p", "", "Name of the plugin to use")
    YourCommand.Flags().StringVarP(&yourCommandOptions.OutputPath, "output", "o", "", "Path to the output file or directory")
    YourCommand.Flags().StringVarP(&yourCommandOptions.Config, "config", "c", "", "Path to configuration file")
    YourCommand.Flags().IntVarP(&yourCommandOptions.Threads, "threads", "j", 1, "Number of concurrent threads to use")
    YourCommand.Flags().BoolP("help", "h", false, "Show help for the command")
}
```