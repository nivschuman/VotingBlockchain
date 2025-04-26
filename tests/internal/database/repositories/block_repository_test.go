package repositories_test

import (
	"math/big"
	"os"
	"testing"

	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	inits "github.com/nivschuman/VotingBlockchain/tests/init"
)

func TestMain(m *testing.M) {
	// === BEFORE ALL TESTS ===
	inits.SetupTests()
	inits.SetupTestsDatabase()

	// Run the tests
	code := m.Run()

	// === AFTER ALL TESTS ===
	inits.CloseTestDatabase()

	// Exit with the right code
	os.Exit(code)
}

func TestGenesisBlock(t *testing.T) {
	inits.ResetTestDatabase()

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
