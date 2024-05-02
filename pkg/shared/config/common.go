package config

import (
	"crypto/tls"
	"time"
)

// BaseHTTPConfig holds common HTTP client configuration settings.
type BaseHTTPConfig struct {
	RetryCount       int
	RetryWaitTime    time.Duration
	RetryMaxWaitTime time.Duration
	Timeout          time.Duration
	TLSClientConfig  *tls.Config
	Proxy            string
}

// RestyHttpClientConfig holds additional configuration settings for the resty http client.
type RestyHttpClientConfig struct {
	BaseHTTPConfig
	Debug bool
}

// General base configuration applicable to all HTTP clients.
func DefaultHttpConfig() BaseHTTPConfig {
	return BaseHTTPConfig{
		RetryCount:       5,
		RetryWaitTime:    1 * time.Second,
		RetryMaxWaitTime: 2 * time.Second,
		Timeout:          10 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12, // Enforce a minimum TLS version
		},
		Proxy: "",
	}
}

// DefaultRestyConfig function returns a specific http config to Resty
func DefaultRestyConfig() RestyHttpClientConfig {
	baseConfig := DefaultHttpConfig()
	return RestyHttpClientConfig{
		BaseHTTPConfig: baseConfig,
		Debug:          false,
	}
}
