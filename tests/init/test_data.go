package test_init

import (
	"time"

	hash "github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	ppk "github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
)

func CreateTestBlock(previousBlockId []byte, transactions []*models.Transaction) (*models.Block, error) {
	minerKeyPair, err := ppk.GenerateKeyPair()

	if err != nil {
		return nil, err
	}

	blockHeader := models.BlockHeader{
		Version:         1,
		PreviousBlockId: previousBlockId,
		MerkleRoot:      make([]byte, 32),
		Timestamp:       time.Now().Unix(),
		NBits:           uint32(0x1d00ffff),
		Nonce:           0,
		MinerPublicKey:  minerKeyPair.PublicKey.AsBytes(),
	}

	blockHeader.SetId()

	for !blockHeader.IsHashBelowTarget() {
		blockHeader.Nonce++
		blockHeader.SetId()
	}

	block := &models.Block{
		Header:       blockHeader,
		Transactions: transactions,
	}

	return block, nil
}

func CreateTestTransaction(govKeyPair *ppk.KeyPair) (*models.Transaction, *ppk.KeyPair, error) {
	voterKeyPair, err := ppk.GenerateKeyPair()

	if err != nil {
		return nil, nil, err
	}

	tx := &models.Transaction{
		Version:        1,
		CandidateId:    1,
		VoterPublicKey: voterKeyPair.PublicKey.AsBytes(),
	}

	tx.SetId()
	signature, err := voterKeyPair.PrivateKey.CreateSignature(tx.Id)

	if err != nil {
		return nil, nil, err
	}

	govSignature, err := govKeyPair.PrivateKey.CreateSignature(hash.HashBytes(tx.VoterPublicKey))

	if err != nil {
		return nil, nil, err
	}

	tx.Signature = signature
	tx.GovernmentSignature = govSignature

	return tx, voterKeyPair, nil
}

func InitializeTestDatabaseWithData(numberOfBlocks int, transactionsPerBlock int) (*ppk.KeyPair, []*models.Block, map[string]*ppk.KeyPair, error) {
	InitializeTestDatabase()
	govKeyPair, err := GenerateTestGovernmentKeyPair()

	if err != nil {
		return nil, nil, nil, err
	}

	genesisBlock := repositories.GlobalBlockRepository.GenesisBlock()
	previousBlockId := genesisBlock.Header.Id

	blocks := make([]*models.Block, numberOfBlocks)
	blocksCounter := 0
	keyPairs := make(map[string]*ppk.KeyPair)

	for range numberOfBlocks {
		blockTransactions := make([]*models.Transaction, transactionsPerBlock)

		for t := range transactionsPerBlock {
			tx, voterKeyPair, err := CreateTestTransaction(govKeyPair)
			keyPairs[string(tx.Id)] = voterKeyPair

			if err != nil {
				return nil, nil, nil, err
			}

			blockTransactions[t] = tx
		}

		block, err := CreateTestBlock(previousBlockId, blockTransactions)

		if err != nil {
			return nil, nil, nil, err
		}

		err = repositories.GlobalBlockRepository.InsertIfNotExists(block)

		if err != nil {
			return nil, nil, nil, err
		}

		previousBlockId = block.Header.Id

		blocks[blocksCounter] = block
		blocksCounter++
	}

	return govKeyPair, blocks, keyPairs, nil
}
