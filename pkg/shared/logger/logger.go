package logger

import (
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

func NewLogger(config *config.Config, name string) hclog.Logger {
	var logLevel hclog.Level

	if config != nil && config.Logger.Level != "" {
		logLevel = getLogLevel(strings.ToUpper(config.Logger.Level))
	} else {
		// env variables has the second priority
		logLevelEnv := os.Getenv("SCANIO_LOG_LEVEL")
		logLevel = getLogLevel(strings.ToUpper(logLevelEnv))
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:        name,
		DisableTime: true,
		Output:      os.Stdout,
		Level:       logLevel,
	})

	return logger
}

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
		return hclog.Info
	}
}
