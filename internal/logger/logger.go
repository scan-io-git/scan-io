package logger

import (
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/internal/config"
)

// NewLogger creates a new hclog.Logger instance based on the YAML configuration and the provided name.
func NewLogger(cfg *config.Config, name string) hclog.Logger {
	logLevel := determineLogLevel(cfg)
	logger := hclog.New(&hclog.LoggerOptions{
		Name:            name,
		DisableTime:     config.GetBoolValue(cfg, "Logger.DisableTime", true),
		JSONFormat:      config.GetBoolValue(cfg, "Logger.JSONFormat", false),
		IncludeLocation: config.GetBoolValue(cfg, "Logger.IncludeLocation", false),
		Output:          os.Stdout,
		Level:           logLevel,
	})
	return logger
}

// determineLogLevel returns a log level determined first by an environment variable, and if not set, by the provided configuration.
// If neither configuration nor environment variable specifies a log level, it defaults to INFO.
func determineLogLevel(cfg *config.Config) hclog.Level {
	if logLevelEnv := os.Getenv("SCANIO_LOG_LEVEL"); logLevelEnv != "" {
		return parseLogLevel(strings.ToUpper(logLevelEnv))
	}
	return parseLogLevel(strings.ToUpper(cfg.Logger.Level))
}

// parseLogLevel converts a string level to hclog.Level.
func parseLogLevel(levelStr string) hclog.Level {
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
		hclog.New(&hclog.LoggerOptions{
			Level:       hclog.Warn,
			DisableTime: true,
			Output:      os.Stdout,
		}).Warn("Unrecognized log level, defaulting to INFO", "providedLevel", levelStr)
		return hclog.Info
	}
}
