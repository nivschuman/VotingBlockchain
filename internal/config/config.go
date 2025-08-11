package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	NetworkConfig    NetworkConfig    `yaml:"network"`
	NodeConfig       NodeConfig       `yaml:"node"`
	GovernmentConfig GovernmentConfig `yaml:"government"`
	MinerConfig      MinerConfig      `yaml:"miner"`
	UiConfig         UiConfig         `yaml:"ui"`
}

var GlobalConfig *Config = nil

func InitializeGlobalConfig(path string) error {
	if GlobalConfig != nil {
		return nil
	}

	var err error
	GlobalConfig, err = LoadConfigFile(path)

	return err
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
