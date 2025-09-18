package repositories_test

import (
	"bytes"
	"os"
	"slices"
	"testing"

	difficulty "github.com/nivschuman/VotingBlockchain/internal/difficulty"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
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

	genesisBlock := inits.TestBlockRepository.GenesisBlock()

	if !genesisBlock.Header.IsHashBelowTarget() {
		t.Fatalf("hash isn't below target")
	}

	cumulativeWork, err := inits.TestBlockRepository.GetBlockCumulativeWork(genesisBlock.Header.Id)

	if err != nil {
		t.Fatalf("failed to get genesis block cumulative work: %v", err)
	}

	if cumulativeWork.Cmp(difficulty.CalculateWork(difficulty.MINIMUM_DIFFICULTY)) != 0 {
		t.Fatalf("genesis block cumulative work is wrong: %s", cumulativeWork.String())
	}
}

func TestActiveChainHeight(t *testing.T) {
	inits.ResetTestDatabase()
	_, _, _, err := inits.CreateTestData(4, 2)
	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	height, err := inits.TestBlockRepository.GetActiveChainHeight()
	if err != nil {
		t.Fatalf("failed to get active chain height: %v", err)
	}

	if height != uint64(4) {
		t.Fatalf("received wrong height %d", height)
	}
}

func TestGetMissingBlockIds(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, _, err := inits.CreateTestData(4, 2)
	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	lastBlock := blocks[len(blocks)-1]

	block1, err := inits.CreateTestBlock(lastBlock.Header.Id, make([]*models.Transaction, 0))
	if err != nil {
		t.Fatalf("failed to create test tx1: %v", err)
	}

	block2, err := inits.CreateTestBlock(lastBlock.Header.Id, make([]*models.Transaction, 0))
	if err != nil {
		t.Fatalf("failed to create test tx1: %v", err)
	}

	ids := structures.NewBytesSet()
	ids.Add(block1.Header.Id)
	ids.Add(block2.Header.Id)
	ids.Add(lastBlock.Header.Id)

	missing, err := inits.TestBlockRepository.GetMissingBlockIds(ids)

	if err != nil {
		t.Fatalf("failed to get missing blocks: %v", err)
	}

	if !missing.Contains(block1.Header.Id) || !missing.Contains(block2.Header.Id) {
		t.Fatalf("missing blocks weren't returned")
	}

	if missing.Contains(lastBlock.Header.Id) {
		t.Fatalf("block that isn't missing was returned")
	}
}

func TestGetBlockCumulativeWork(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, _, err := inits.CreateTestData(4, 2)
	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	expectedCumulativeWork := inits.TestBlockRepository.GenesisBlock().GetBlockWork()
	for _, block := range blocks {
		expectedCumulativeWork.Add(block.GetBlockWork(), expectedCumulativeWork)
	}

	lastBlock := blocks[len(blocks)-1]
	cumulativeWork, err := inits.TestBlockRepository.GetBlockCumulativeWork(lastBlock.Header.Id)

	if err != nil {
		t.Fatalf("failed to get block cumulative work: %v", err)
	}

	if expectedCumulativeWork.Cmp(cumulativeWork) != 0 {
		t.Fatalf("cumulative work isn't as expected")
	}
}

func TestGetMedianTimePast(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, _, err := inits.CreateTestData(4, 2)
	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	times := make([]int64, len(blocks))
	for idx, block := range blocks {
		times[idx] = block.Header.Timestamp
	}

	slices.Sort(times)
	expectedMedianTime := times[len(times)/2]

	lastBlock := blocks[len(blocks)-1]
	medianTime, err := inits.TestBlockRepository.GetMedianTimePast(lastBlock.Header.Id, 4)

	if err != nil {
		t.Fatalf("failed to calculate median time: %v", err)
	}

	if expectedMedianTime != medianTime {
		t.Fatalf("calculated wrong median time")
	}
}

func TestGetBlockLocator(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, _, err := inits.CreateTestData(19, 1)
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	genesisBlock := inits.TestBlockRepository.GenesisBlock()
	blocks = append([]*models.Block{genesisBlock}, blocks...)

	lastBlock := blocks[len(blocks)-1]
	locator, err := inits.TestBlockRepository.GetBlockLocator(lastBlock.Header.Id)
	if err != nil {
		t.Fatalf("get block locator failed: %v", err)
	}

	expectedHeights := []int{
		19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 8, 4, 0,
	}

	if locator.Length() != len(expectedHeights) {
		t.Fatalf("expected locator length %d, got %d", len(expectedHeights), locator.Length())
	}

	for i, expectedHeight := range expectedHeights {
		expectedId := blocks[expectedHeight].Header.Id
		locatorId := locator.Get(i)
		if !bytes.Equal(locatorId, expectedId) {
			t.Errorf("locator[%d]: expected block at height %d (id %x), got id %x", i, expectedHeight, expectedId, locatorId)
		}
	}
}

func TestGetActiveChainBlockLocator(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, _, err := inits.CreateTestData(19, 1)
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	genesisBlock := inits.TestBlockRepository.GenesisBlock()
	blocks = append([]*models.Block{genesisBlock}, blocks...)

	locator, err := inits.TestBlockRepository.GetActiveChainBlockLocator()
	if err != nil {
		t.Fatalf("get block locator failed: %v", err)
	}

	expectedHeights := []int{
		19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 8, 4, 0,
	}

	if locator.Length() != len(expectedHeights) {
		t.Fatalf("expected locator length %d, got %d", len(expectedHeights), locator.Length())
	}

	for i, expectedHeight := range expectedHeights {
		expectedId := blocks[expectedHeight].Header.Id
		locatorId := locator.Get(i)
		if !bytes.Equal(locatorId, expectedId) {
			t.Errorf("locator[%d]: expected block at height %d (id %x), got id %x", i, expectedHeight, expectedId, locatorId)
		}
	}
}

func TestGetNextBlockIds(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, _, err := inits.CreateTestData(20, 1)
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	locator := structures.NewBlockLocator()
	for i := 10; i >= 0; i-- {
		locator.Add(blocks[i].Header.Id)
	}

	stopHash := []byte(nil)
	limit := 5

	blockIds, err := inits.TestBlockRepository.GetNextBlocksIds(locator, stopHash, limit)
	if err != nil {
		t.Fatalf("get next block ids failed: %v", err)
	}

	if blockIds.Length() != limit {
		t.Fatalf("expected %d blocks but got %d", limit, blockIds.Length())
	}

	for i := range limit {
		expectedId := blocks[11+i].Header.Id

		if !blockIds.Contains(expectedId) {
			t.Fatalf("missing block %x", expectedId)
		}
	}
}

func TestGetNextWorkRequired(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, _, err := inits.CreateTestData(10, 1)
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	//next block is not at difficulty adjustment level
	lastBlock := blocks[7] // height 8
	nBits, err := inits.TestBlockRepository.GetNextWorkRequired(lastBlock.Header.Id)
	if err != nil {
		t.Fatalf("GetNextWorkRequired error: %v", err)
	}

	if nBits != lastBlock.Header.NBits {
		t.Fatalf("Expected nBits %08x, got %08x", lastBlock.Header.NBits, nBits)
	}

	//next block is at difficulty adjustment level
	lastBlock = blocks[8] // height 9
	nBits, err = inits.TestBlockRepository.GetNextWorkRequired(lastBlock.Header.Id)
	if err != nil {
		t.Fatalf("GetNextWorkRequired error: %v", err)
	}

	if nBits == lastBlock.Header.NBits {
		t.Logf("Difficulty did not change at interval (could be expected if timestamps are perfect)")
	} else {
		t.Logf("Difficulty adjusted at interval: new nBits %08x", nBits)
	}
}
