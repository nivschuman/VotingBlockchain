package mining_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	mining "github.com/nivschuman/VotingBlockchain/internal/mining"
	data_models "github.com/nivschuman/VotingBlockchain/internal/models"
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

func TestCreateBlockTemplate(t *testing.T) {
	inits.ResetTestDatabase()

	govKeyPair, blocks, _, err := inits.CreateTestData(2, 2)
	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	tx1, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("failed to create test tx1: %v", err)
	}

	err = inits.TestTransactionRepository.InsertIfNotExists(tx1)
	if err != nil {
		t.Fatalf("failed to insert test tx1: %v", err)
	}

	tx2, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("failed to create test tx2: %v", err)
	}

	err = inits.TestTransactionRepository.InsertIfNotExists(tx2)
	if err != nil {
		t.Fatalf("failed to insert test tx2: %v", err)
	}

	getNetworkTime := func() int64 { return time.Now().Unix() }
	minerProps := mining.MinerProperties{
		NodeVersion:    inits.TestConfig.NodeConfig.Version,
		MinerPublicKey: inits.TestConfig.GovernmentConfig.PublicKey,
	}
	miner := mining.NewMinerImpl(getNetworkTime, inits.TestBlockRepository, inits.TestTransactionRepository, minerProps)

	template, err := miner.CreateBlockTemplate()
	if err != nil {
		t.Fatalf("failed to create block template: %v", err)
	}

	lastBlock := blocks[len(blocks)-1]
	if !bytes.Equal(lastBlock.Header.Id, template.Header.PreviousBlockId) {
		t.Fatalf("previous block isn't last block in active chain")
	}

	if len(template.Transactions) != 2 {
		t.Fatalf("template doesn't contain right amount of transactions")
	}

	if !bytes.Equal(tx1.Id, template.Transactions[0].Id) && !bytes.Equal(tx1.Id, template.Transactions[1].Id) {
		t.Fatalf("tx1 isn't in block template")
	}

	if !bytes.Equal(tx2.Id, template.Transactions[0].Id) && !bytes.Equal(tx2.Id, template.Transactions[1].Id) {
		t.Fatalf("tx2 isn't in block template")
	}
}

func TestMineBlockTemplate(t *testing.T) {
	inits.ResetTestDatabase()

	govKeyPair, blocks, _, err := inits.CreateTestData(2, 2)
	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	tx1, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("failed to create test tx1: %v", err)
	}

	lastBlock := blocks[len(blocks)-1]
	template, err := inits.CreateTestBlock(lastBlock.Header.Id, []*data_models.Transaction{tx1})
	template.Header.Nonce = 0

	if err != nil {
		t.Fatalf("failed to create test block: %v", err)
	}

	getNetworkTime := func() int64 { return time.Now().Unix() }
	minerProps := mining.MinerProperties{
		NodeVersion:    inits.TestConfig.NodeConfig.Version,
		MinerPublicKey: inits.TestConfig.GovernmentConfig.PublicKey,
	}
	miner := mining.NewMinerImpl(getNetworkTime, inits.TestBlockRepository, inits.TestTransactionRepository, minerProps)

	checkBlock := func(block *data_models.Block) {
		t.Logf("mined block nonce is %d", block.Header.Nonce)

		if !block.Header.IsHashBelowTarget() {
			t.Fatalf("mined block hash isn't valid")
		}

		if !bytes.Equal(lastBlock.Header.Id, block.Header.PreviousBlockId) {
			t.Fatalf("mined block previous block is wrong")
		}
	}

	miner.AddHandler(checkBlock)
	miner.MineBlockTemplate(template)
}

func BenchmarkMineBlockTemplate(b *testing.B) {
	inits.ResetTestDatabase()

	govKeyPair, blocks, _, err := inits.CreateTestData(2, 2)
	if err != nil {
		b.Fatalf("failed to setup test data: %v", err)
	}

	tx1, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		b.Fatalf("failed to create test tx1: %v", err)
	}

	lastBlock := blocks[len(blocks)-1]

	getNetworkTime := func() int64 { return time.Now().Unix() }
	minerProps := mining.MinerProperties{
		NodeVersion:    inits.TestConfig.NodeConfig.Version,
		MinerPublicKey: inits.TestConfig.GovernmentConfig.PublicKey,
	}
	miner := mining.NewMinerImpl(getNetworkTime, inits.TestBlockRepository, inits.TestTransactionRepository, minerProps)

	template, err := inits.CreateTestBlock(lastBlock.Header.Id, []*data_models.Transaction{tx1})
	if err != nil {
		b.Fatalf("failed to create test block: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		template.Header.Version = int32(i)
		template.Header.Nonce = 0
		template.Header.NBits = 0x1e7fffff
		miner.MineBlockTemplate(template)
	}
}
