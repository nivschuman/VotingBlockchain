package repositories_test

import (
	"math/big"
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

	cumulativeWork, err := repositories.GlobalBlockRepository.GetBlockCumulativeWork(genesisBlock.Header.Id)

	if err != nil {
		t.Fatalf("failed to get genesis block cumulative work: %v", err)
	}

	if cumulativeWork.Cmp(big.NewInt(4295032833)) != 0 {
		t.Fatalf("genesis block cumulative work is wrong: %s", cumulativeWork.String())
	}
}
