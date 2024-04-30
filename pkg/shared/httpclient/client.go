package httpclient

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-hclog"
	"github.com/scan-io-git/scan-io/pkg/shared/config"
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
func applyHttpClientConfig(httpConfig *config.HttpClient) config.RestyHttpClientConfig {
	var cfg config.RestyHttpClientConfig

	//TODO check time.duration values * with time.Second
	if httpConfig != nil {
		cfg.Debug = config.GetBoolValue(httpConfig, "Debug", config.DefaultRestyConfig().Debug)
		cfg.RetryCount = config.SetThen(httpConfig.RetryCount, config.DefaultRestyConfig().RetryCount)
		cfg.RetryWaitTime = config.SetThen(httpConfig.RetryWaitTime, config.DefaultRestyConfig().RetryWaitTime)
		cfg.RetryMaxWaitTime = config.SetThen(httpConfig.RetryMaxWaitTime, config.DefaultRestyConfig().RetryMaxWaitTime)
		cfg.Timeout = config.SetThen(httpConfig.Timeout, config.DefaultRestyConfig().Timeout)
		cfg.TLSClientConfig.InsecureSkipVerify = !config.GetBoolValue(httpConfig.TlsClientConfig, "Verify", true)

		if httpConfig.Proxy.Host != "" && httpConfig.Proxy.Port != "" {
			cfg.Proxy = fmt.Sprintf("%s:%s", httpConfig.Proxy.Host, httpConfig.Proxy.Port)
		}
	}

	return cfg
}
