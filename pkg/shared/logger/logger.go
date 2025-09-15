package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

type Close func() error

// NewLogger creates a new hclog.Logger instance based on the YAML configuration and the provided name.
func NewLogger(cfg *config.Config, name string) (hclog.Logger, Close, error) {
	logLevel := determineLogLevel(cfg)

	out, cleanup, err := buildOutputs(cfg)
	if err != nil {
		out = os.Stderr
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:            name,
		DisableTime:     config.SetThenPtr(cfg.Logger.DisableTime, true),
		JSONFormat:      config.SetThenPtr(cfg.Logger.JSONFormat, false),
		IncludeLocation: config.SetThenPtr(cfg.Logger.IncludeLocation, false),
		Level:           logLevel,
		Output:          out,
	})
	return logger, cleanup, err
}

func buildOutputs(cfg *config.Config) (io.Writer, Close, error) {
	writers := []io.Writer{os.Stderr}

	filePath := fmt.Sprintf("%s/%s", strings.TrimSpace(cfg.Logger.FolderPath), "scanio.log")

	if filePath == "" {
		return writers[0], func() error { return nil }, nil
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return writers[0], func() error { return nil }, fmt.Errorf("create log dir: %w", err)
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return writers[0], func() error { return nil }, fmt.Errorf("open log file %q: %w", filePath, err)
	}

	writers = append(writers, f)
	return io.MultiWriter(writers...), f.Close, nil
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

// GetLoggerOutput prepares the logger output.
func GetLoggerOutput(logger hclog.Logger) io.Writer {
	return logger.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	})
}

func ForkLogger(base hclog.Logger, name string) hclog.Logger {
	return base.ResetNamed(name)
}
