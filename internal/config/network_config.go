package config

type NetworkConfig struct {
	Ip   string `yaml:"ip"`
	Port uint16 `yaml:"port"`
}
