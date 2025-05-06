package config_test

import (
	"os"
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/config"
	inits "github.com/nivschuman/VotingBlockchain/tests/init"
)

func TestMain(m *testing.M) {
	// === BEFORE ALL TESTS ===
	inits.SetupTests()

	// Run the tests
	code := m.Run()

	// === AFTER ALL TESTS ===

	// Exit with the right code
	os.Exit(code)
}

func TestGlobalConfig(t *testing.T) {
	conf := config.GlobalConfig

	if conf.NodeConfig.Version != 1 {
		t.Fatalf("Node version wasn't set correctly, is %d", conf.NodeConfig.Version)
	}

	if conf.NodeConfig.Type != 1 {
		t.Fatalf("Node type wasn't set correctly, is %d", conf.NodeConfig.Version)
	}
}
