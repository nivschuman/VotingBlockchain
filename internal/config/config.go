package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	NetworkConfig `yaml:"network"`
	NodeConfig    `yaml:"node"`
}

var GlobalConfig *Config = nil

func InitializeGlobalConfig() error {
	if GlobalConfig != nil {
		return nil
	}

	var err error
	GlobalConfig, err = LoadAppConfig()

	return err
}

func LoadAppConfig() (*Config, error) {
	env := os.Getenv("APP_ENV")

	if env == "" {
		return nil, errors.New("APP_ENV environment variable not set")
	}

	configFile := fmt.Sprintf("config/config-%s.yml", env)
	return LoadConfigFile(configFile)
}

func LoadConfigFile(path string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)

	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}
