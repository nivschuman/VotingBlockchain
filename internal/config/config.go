package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	NetworkConfig `yaml:"network"`
	NodeConfig    `yaml:"node"`
}

var GlobalConfig *Config = LoadGlobalConfig()

func LoadGlobalConfig() *Config {
	env := os.Getenv("APP_ENV")

	if env == "" {
		log.Fatal("APP_ENV environment variable not set")
	}

	configFile := fmt.Sprintf("config/config-%s.yml", env)
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	return config
}

func LoadConfig(path string) (*Config, error) {
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
