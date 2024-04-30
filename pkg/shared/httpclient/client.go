package httpclient

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

type (
	RestyConfig struct {
		Debug            bool
		RetryCount       int
		RetryWaitTime    time.Duration
		RetryMaxWaitTime time.Duration
		Timeout          time.Duration
		// Certificates     tls.Certificate
		// RootCertificate  string
		TLSClientConfig *tls.Config
		Proxy           string
		Logger          hclog.Logger
	}
)

// HclogAdapter adapts an hclog.Logger to be compatible with the resty log.Logger interface.
type HclogAdapter struct {
	logger hclog.Logger
}

// NewHclogAdapter creates a new adapter that will forward messages to a hclog.Logger.
func NewHclogAdapter(logger hclog.Logger) resty.Logger {
	return &HclogAdapter{logger: logger}
}

// // Write implements the io.Writer interface which is required by log.Logger.
// func (a *HclogAdapter) Write(p []byte) (n int, err error) {
// 	a.logger.Info(string(p))
// 	return len(p), nil
// }

// Errorf logs a message at error level.
func (a *HclogAdapter) Errorf(format string, v ...interface{}) {
	a.logger.Error(fmt.Sprintf(format, v...))
}

// Warnf logs a message at warning level.
func (a *HclogAdapter) Warnf(format string, v ...interface{}) {
	a.logger.Warn(fmt.Sprintf(format, v...))
}

// Infof logs a message at info level.
func (a *HclogAdapter) Infof(format string, v ...interface{}) {
	a.logger.Info(fmt.Sprintf(format, v...))
}

// Debugf logs a message at debug level.
func (a *HclogAdapter) Debugf(format string, v ...interface{}) {
	a.logger.Debug(fmt.Sprintf(format, v...))
}

// SetLoggerForResty sets the adapted hclog.Logger as the logger for Resty.
func SetLoggerForResty(client *resty.Client, logger hclog.Logger) {
	client.SetLogger(NewHclogAdapter(logger))
}

// DefaultConfig provides a default configuration for the resty client, used when no specific settings are provided.
func defaultConfig() RestyConfig {
	return RestyConfig{
		Debug:            false,
		RetryCount:       5,
		RetryWaitTime:    1 * time.Second,
		RetryMaxWaitTime: 2 * time.Second,
		Timeout:          10 * time.Second,
		TLSClientConfig:  &tls.Config{}, // TODO add safe defaults. the default is not really safe in terms of tls versions and cipher suites
		Proxy:            "",
	}
}

// InitializeRestyClient initializes and configures a resty client based on the provided configuration.
func InitializeRestyClient(logger hclog.Logger, cfg *config.Config) *resty.Client {
	client := resty.New()
	if logger != nil {
		SetLoggerForResty(client, logger)
	}

	// Apply the configuration settings from the config file or use defaults
	restyConfig := applyHttpClientConfig(&cfg.HttpClient)
	client.
		SetDebug(restyConfig.Debug).
		SetRetryCount(restyConfig.RetryCount).
		SetRetryWaitTime(restyConfig.RetryWaitTime).
		SetRetryMaxWaitTime(restyConfig.RetryMaxWaitTime).
		SetTimeout(restyConfig.Timeout).
		SetTLSClientConfig(restyConfig.TLSClientConfig).
		SetProxy(restyConfig.Proxy)

	return client
}

// applyHttpClientConfig applies the HttpClient configuration or uses default values.
func applyHttpClientConfig(httpConfig *config.HttpClient) RestyConfig {
	config := defaultConfig()

	if httpConfig != nil {
		if httpConfig.Debug != nil {
			config.Debug = *httpConfig.Debug
		}
		if httpConfig.RetryCount != 0 {
			config.RetryCount = httpConfig.RetryCount
		}
		if httpConfig.RetryWaitTime != 0 {
			config.RetryWaitTime = httpConfig.RetryWaitTime
		}
		if httpConfig.RetryMaxWaitTime != 0 {
			config.RetryMaxWaitTime = httpConfig.RetryMaxWaitTime
		}
		if httpConfig.Timeout != 0 {
			config.Timeout = httpConfig.Timeout
		}
		if httpConfig.TlsClientConfig.Verify != nil {
			config.TLSClientConfig.InsecureSkipVerify = !*httpConfig.TlsClientConfig.Verify
		}
		if httpConfig.Proxy.Host != "" && httpConfig.Proxy.Port != "" {
			config.Proxy = fmt.Sprintf("%s:%s", httpConfig.Proxy.Host, httpConfig.Proxy.Port)
		}
	}

	return config
}
