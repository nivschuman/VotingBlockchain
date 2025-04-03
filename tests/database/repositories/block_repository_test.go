package repositories_test

import (
	"testing"

	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	inits "github.com/nivschuman/VotingBlockchain/tests/init"
)

func TestGenesisBlock(t *testing.T) {
	inits.InitializeTestDatabase()

	genesisBlock := repositories.GlobalBlockRepository.GenesisBlock()

	if !genesisBlock.Header.IsHashBelowTarget() {
		t.Fatalf("hash isn't below target")
	}

	err := repositories.GlobalBlockRepository.InsertBlock(genesisBlock)

	if err != nil {
		t.Fatalf("failed to insert genesis block: %v", err)
	}
}
