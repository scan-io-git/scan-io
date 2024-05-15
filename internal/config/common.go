package config

import (
	"crypto/tls"
	"time"
)

// BaseHTTPConfig holds common HTTP client configuration settings.
type BaseHTTPConfig struct {
	RetryCount       int           // Number of retries for failed requests
	RetryWaitTime    time.Duration // Wait time between retries
	RetryMaxWaitTime time.Duration // Maximum wait time for retries
	Timeout          time.Duration // Timeout for requests
	TLSClientConfig  *tls.Config   // TLS configuration
	Proxy            string        // Proxy address
}

// RestyHTTPClientConfig holds additional configuration settings for the Resty HTTP client.
type RestyHTTPClientConfig struct {
	BaseHTTPConfig
	Debug bool // Flag to enable Resty debug mode
}

// DefaultHTTPConfig returns a base configuration for HTTP clients with default values.
func DefaultHTTPConfig() BaseHTTPConfig {
	return BaseHTTPConfig{
		RetryCount:       5,
		RetryWaitTime:    1 * time.Second,
		RetryMaxWaitTime: 5 * time.Second,
		Timeout:          30 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12, // Enforce a minimum TLS version
			InsecureSkipVerify: false,            // Ensure TLS certificates are verified
		},
		Proxy: "", // No proxy by default
	}
}

// DefaultRestyConfig returns a default configuration for the Resty HTTP client, extending the base HTTP configuration.
func DefaultRestyConfig() RestyHTTPClientConfig {
	baseConfig := DefaultHTTPConfig()
	return RestyHTTPClientConfig{
		BaseHTTPConfig: baseConfig,
		Debug:          false, // Debug mode is disabled by default
	}
}
