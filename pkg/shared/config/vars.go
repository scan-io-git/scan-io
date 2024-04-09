package config

import (
	"time"
)

type Config struct {
	Logger     Logger     `yaml:"logger"`
	HttpClient HttpClient `yaml:"http_client"`
}

type Logger struct {
	Level string `yaml:"level"`
}

type HttpClient struct {
	Debug            string          `yaml:"debug"`
	RetryCount       time.Duration   `yaml:"retry_count"`
	RetryWaitTime    time.Duration   `yaml:"retry_wait_time"`
	RetryMaxWaitTime time.Duration   `yaml:"retry_max_wait_time"`
	Timeout          time.Duration   `yaml:"timeout"`
	TlsClientConfig  TlsClientConfig `yaml:"tls_client_config"`
	Proxy            Proxy           `yaml:"proxy"`
}

type TlsClientConfig struct {
	Verify bool `yaml:"verify"`
}

type Proxy struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}
