package config

import (
	"encoding/hex"
)

type GovernmentConfig struct {
	PublicKey []byte `yaml:"public-key"`
}

func (g *GovernmentConfig) UnmarshalYAML(unmarshal func(any) error) error {
	var raw struct {
		PublicKey string `yaml:"public-key"`
	}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	publicKeyBytes, err := hex.DecodeString(raw.PublicKey)
	if err != nil {
		return err
	}

	g.PublicKey = publicKeyBytes
	return nil
}
