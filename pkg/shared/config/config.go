package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
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

	//TODO: move to default internal configuration
	// if len(configPath) == 0 {

	// }

	if err := LoadYAML(configPath, &config); err != nil {
		return nil, err
	}

	return config, nil
}
