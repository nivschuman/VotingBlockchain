package config_test

import (
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/config"
	_ "github.com/nivschuman/VotingBlockchain/tests/init"
)

func TestGlobalConfig(t *testing.T) {
	conf := config.GlobalConfig

	if conf.NodeConfig.Version != 1 {
		t.Fatalf("Node version wasn't set correctly, is %d", conf.NodeConfig.Version)
	}

	if conf.NodeConfig.Type != 1 {
		t.Fatalf("Node type wasn't set correctly, is %d", conf.NodeConfig.Version)
	}
}
