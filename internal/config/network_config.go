package config

import (
	"net"

	"gopkg.in/yaml.v2"
)

type NetworkConfig struct {
	Ip                     net.IP `yaml:"ip"`
	Port                   uint16 `yaml:"port"`
	PingInterval           uint32 `yaml:"ping-interval"`
	PongTimeout            uint32 `yaml:"pong-timeout"`
	SendDataInterval       uint32 `yaml:"send-data-interval"`
	GetAddrInterval        uint32 `yaml:"get-addr-interval"`
	MaxNumberOfConnections int    `yaml:"max-number-of-connections"`
	AddressesFile          string `yaml:"addresses-file"`
}

func (n *NetworkConfig) UnmarshalYAML(unmarshal func(any) error) error {
	var raw struct {
		Ip                     string `yaml:"ip"`
		Port                   uint16 `yaml:"port"`
		PingInterval           uint32 `yaml:"ping-interval"`
		PongTimeout            uint32 `yaml:"pong-timeout"`
		SendDataInterval       uint32 `yaml:"send-data-interval"`
		GetAddrInterval        uint32 `yaml:"get-addr-interval"`
		MaxNumberOfConnections int    `yaml:"max-number-of-connections"`
		AddressesFile          string `yaml:"addresses-file"`
	}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	n.Ip = net.ParseIP(raw.Ip)
	if n.Ip == nil {
		return &yaml.TypeError{Errors: []string{"invalid IP address"}}
	}

	n.Port = raw.Port
	n.PingInterval = raw.PingInterval
	n.PongTimeout = raw.PongTimeout
	n.SendDataInterval = raw.SendDataInterval
	n.GetAddrInterval = raw.GetAddrInterval
	n.MaxNumberOfConnections = raw.MaxNumberOfConnections
	n.AddressesFile = raw.AddressesFile

	return nil
}
