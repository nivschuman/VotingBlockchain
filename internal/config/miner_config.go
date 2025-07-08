package config

import (
	"encoding/hex"
)

type MinerConfig struct {
	PublicKey []byte `yaml:"public-key"`
	Enabled   bool   `yaml:"enabled"`
}

func (m *MinerConfig) UnmarshalYAML(unmarshal func(any) error) error {
	var raw struct {
		PublicKey string `yaml:"public-key"`
		Enabled   bool   `yaml:"enabled"`
	}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	publicKeyBytes, err := hex.DecodeString(raw.PublicKey)
	if err != nil {
		return err
	}

	m.PublicKey = publicKeyBytes
	m.Enabled = raw.Enabled
	return nil
}
