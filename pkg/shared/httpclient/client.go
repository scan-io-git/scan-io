package httpclient

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-hclog"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
)

// Client represents a wrapper for the Resty client.
type Client struct {
	RestyClient *resty.Client
}

// CustomRoundTripper adds custom headers to every request and delegates to the base transport.
type CustomRoundTripper struct {
	BaseTransport http.RoundTripper
	Headers       map[string]string
}

// HclogAdapter adapts an hclog.Logger to be compatible with the resty log.Logger interface.
type HclogAdapter struct {
	logger hclog.Logger
}

// NewHclogAdapter creates a new adapter that forwards messages to an hclog.Logger.
func NewHclogAdapter(logger hclog.Logger) resty.Logger {
	return &HclogAdapter{logger: logger}
}

// Errorf logs a message at the error level.
func (a *HclogAdapter) Errorf(format string, v ...interface{}) {
	a.logger.Error(fmt.Sprintf(format, v...))
}

// Warnf logs a message at the warning level.
func (a *HclogAdapter) Warnf(format string, v ...interface{}) {
	a.logger.Warn(fmt.Sprintf(format, v...))
}

// Infof logs a message at the info level.
func (a *HclogAdapter) Infof(format string, v ...interface{}) {
	a.logger.Info(fmt.Sprintf(format, v...))
}

// Debugf logs a message at the debug level.
func (a *HclogAdapter) Debugf(format string, v ...interface{}) {
	a.logger.Debug(fmt.Sprintf(format, v...))
}

// SetLoggerForResty sets the adapted hclog.Logger as the logger for Resty.
func SetLoggerForResty(client *resty.Client, logger hclog.Logger) {
	client.SetLogger(NewHclogAdapter(logger))
}

// New creates and initializes a new Resty client based on the provided configuration.
func New(logger hclog.Logger, cfg *config.Config) (*Client, error) {
	client := resty.New()
	if logger != nil {
		SetLoggerForResty(client, logger)
	}

	// Apply the configuration settings from the config file or use defaults
	restyConfig := applyHTTPClientConfig(&cfg.HTTPClient)

	client.
		SetDebug(restyConfig.Debug).
		SetRetryCount(restyConfig.RetryCount).
		SetRetryWaitTime(restyConfig.RetryWaitTime).
		SetRetryMaxWaitTime(restyConfig.RetryMaxWaitTime).
		SetTimeout(restyConfig.Timeout).
		SetTLSClientConfig(restyConfig.TLSClientConfig)

	if restyConfig.Proxy != "" {
		client.SetProxy(restyConfig.Proxy)
	}

	// Apply custom headers if they are defined in the configuration
	if len(cfg.HTTPClient.CustomHeaders) > 0 {
		for key, value := range cfg.HTTPClient.CustomHeaders {
			client.SetHeader(key, value)
		}
	}
	return &Client{RestyClient: client}, nil
}

// applyHTTPClientConfig applies the HTTPClient configuration or uses default values.
func applyHTTPClientConfig(httpConfig *config.HTTPClient) config.RestyHTTPClientConfig {
	defaultCfg := config.DefaultRestyConfig()
	cfg := defaultCfg

	// TODO: Add handling debug via the logger config
	cfg.Debug = config.SetThenPtr(httpConfig.Debug, defaultCfg.Debug)
	cfg.RetryCount = config.SetThen(httpConfig.RetryCount, defaultCfg.RetryCount)
	cfg.RetryWaitTime = config.SetThen(httpConfig.RetryWaitTime, defaultCfg.RetryWaitTime)
	cfg.RetryMaxWaitTime = config.SetThen(httpConfig.RetryMaxWaitTime, defaultCfg.RetryMaxWaitTime)
	cfg.Timeout = config.SetThen(httpConfig.Timeout, defaultCfg.Timeout)
	cfg.TLSClientConfig.InsecureSkipVerify = !config.SetThenPtr(
		httpConfig.TLSClientConfig.Verify, defaultCfg.TLSClientConfig.InsecureSkipVerify)

	// TODO: Use default value from default config
	if httpConfig.Proxy.Host != "" && httpConfig.Proxy.Port != 0 {
		cfg.Proxy = fmt.Sprintf("%s:%d", httpConfig.Proxy.Host, httpConfig.Proxy.Port)
	}

	return cfg
}

// RoundTrip executes a single HTTP transaction and adds custom headers.
func (rt *CustomRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range rt.Headers {
		req.Header.Set(key, value)
	}
	return rt.BaseTransport.RoundTrip(req)
}
