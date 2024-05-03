package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ValidateConfig checks if the global configurations have valid values.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("YAML global config validation: configuration object is nil")
	}
	if err := HttpGlobalConfigValidation(&cfg.HttpClient); err != nil {
		return fmt.Errorf("YAML global config validation: http_client directive is invalid. %v", err)
	}
	return nil
}

// HttpGlobalConfigValidation checks if the HTTP configurations have valid values.
func HttpGlobalConfigValidation(httpConfig *HttpClient) error {
	if httpConfig == nil {
		return fmt.Errorf("http configuration is nil")
	}

	durations := map[string]time.Duration{
		"RetryMaxWaitTime": httpConfig.RetryMaxWaitTime,
		"RetryWaitTime":    httpConfig.RetryWaitTime,
		"Timeout":          httpConfig.Timeout,
	}

	if httpConfig.RetryCount < 0 || httpConfig.RetryCount > 20 {
		return fmt.Errorf("retry_count must be between 1 and 20: %v", httpConfig.RetryCount)
	}

	for name, duration := range durations {
		if err := validateDuration(duration); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if duration > 100*time.Second {
			return fmt.Errorf("%s: duration is too long - %v", name, duration)
		}
	}

	if err := validateProxy(&httpConfig.Proxy); err != nil {
		return err
	}

	return nil
}

// validateDuration checks that a time.Duration is not negative.
func validateDuration(d time.Duration) error {
	if d < 0 {
		return fmt.Errorf("invalid duration: %v cannot be negative", d)
	}
	return nil
}

// ValidateProxy checks if the given Proxy settings are valid.
func validateProxy(proxy *Proxy) error {
	if proxy == nil {
		return fmt.Errorf("proxy configuration is nil")
	}

	// If host or port is not set, skip further validation
	if proxy.Host == "" || proxy.Port == 0 {
		return nil
	}

	if err := validateHost(&proxy.Host); err != nil {
		return err
	}

	if err := validatePort(proxy.Port); err != nil {
		return err
	}

	return nil
}

// validateHost checks if the host part of the proxy configuration is valid.
// It expects a pointer to the host string so it can modify the global config directly if necessary.
// It ensures the host includes a scheme; adds "http" if missing.
func validateHost(host *string) error {
	if host == nil {
		return fmt.Errorf("host string pointer is nil")
	}

	if !strings.Contains(*host, "://") {
		*host = "http://" + *host
	}
	*host = strings.TrimRight(*host, "/")

	// TODO add domain or IP validation
	_, err := url.Parse(*host)
	if err != nil {
		return fmt.Errorf("invalid host URL: %w", err)
	}

	return nil
}

// validatePort checks if the port part of the proxy configuration is valid.
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}
