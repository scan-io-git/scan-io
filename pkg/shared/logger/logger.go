package logger

import (
	"os"
	"reflect"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// NewLogger creates a new hclog.Logger instance based on the yml configuration and the provided name.
func NewLogger(config *config.Config, name string) hclog.Logger {
	logLevel := determineLogLevel(config)
	logger := hclog.New(&hclog.LoggerOptions{
		Name:            name,
		DisableTime:     getBoolValue(config, "DisableTime", true),
		JSONFormat:      getBoolValue(config, "JSONFormat", false),
		IncludeLocation: getBoolValue(config, "IncludeLocation", false),
		Output:          os.Stdout,
		Level:           logLevel,
	})
	return logger
}

// determineLogLevel return a log level which is determined first by the configuration provided, and if not set, by an environment variable.
// If neither configuration nor environment variable specifies a log level, it defaults to INFO.
func determineLogLevel(config *config.Config) hclog.Level {
	if config != nil && config.Logger.Level != "" {
		return getLogLevel(strings.ToUpper(config.Logger.Level))
	}
	return getLogLevel(strings.ToUpper(os.Getenv("SCANIO_LOG_LEVEL")))
}

// getBoolValue retrieves a boolean value based on the specified field from the LoggerConfig struct.
// It uses the provided defaultValue if the specific boolean field is not explicitly set.
func getBoolValue(config *config.Config, field string, defaultValue bool) bool {
	if config == nil {
		return defaultValue
	}

	val := reflect.ValueOf(config.Logger)
	valueField := val.FieldByName(field)

	if valueField.IsValid() && !valueField.IsNil() {
		return valueField.Elem().Bool()
	}
	return defaultValue
}

// getLogLevel converts a string level to hclog.Level.
func getLogLevel(levelStr string) hclog.Level {
	switch levelStr {
	case "TRACE":
		return hclog.Trace
	case "DEBUG":
		return hclog.Debug
	case "INFO":
		return hclog.Info
	case "WARN":
		return hclog.Warn
	case "ERROR":
		return hclog.Error
	default:
		hclog.New(&hclog.LoggerOptions{Level: hclog.Warn, DisableTime: true, Output: os.Stdout}).
			Warn("Unrecognized log level, defaulting to INFO", "providedLevel", levelStr)
		return hclog.Info
	}
}
