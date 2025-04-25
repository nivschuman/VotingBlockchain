package config

type NetworkConfig struct {
	Ip               string `yaml:"ip"`
	Port             uint16 `yaml:"port"`
	PingInterval     uint32 `yaml:"ping-interval"`
	PongTimeout      uint32 `yaml:"pong-timeout"`
	SendDataInterval uint32 `yaml:"send-data-interval"`
}
