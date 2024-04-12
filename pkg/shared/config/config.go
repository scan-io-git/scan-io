package config

import (
	"fmt"
	"os"
	"time"

	yaml "gopkg.in/yaml.v2"
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

func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a file", path)
	}
	return nil
}

func LoadYAML(configPath string, data interface{}) error {
	if err := ValidateConfigPath(configPath); err != nil {
		return err
	}

	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(data); err != nil {
		return err
	}

	return nil
}

func NewConfig(configPath string) (*Config, error) {
	config := &Config{}

	if err := LoadYAML(configPath, &config); err != nil {
		return nil, err
	}

	return config, nil
}
