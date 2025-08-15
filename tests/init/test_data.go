package test_init

import (
	"fmt"
	"log"
	"time"

	hash "github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	ppk "github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
	"github.com/nivschuman/VotingBlockchain/internal/difficulty"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
	"github.com/nivschuman/VotingBlockchain/internal/voters"
)

func PrintTestVotersAndGovernmentKeyPair(gov *ppk.KeyPair, voters []*voters.Voter) error {
	govPub := gov.PublicKey.AsBytes()
	govPriv, err := gov.PrivateKey.AsBytes()
	if err != nil {
		return nil
	}

	log.Printf("Government Private Key: %x\n", govPriv)
	log.Printf("Government Public Key:  %x\n\n", govPub)

	for _, v := range voters {
		vPub := v.KeyPair.PublicKey.AsBytes()
		vPriv, err := v.KeyPair.PrivateKey.AsBytes()
		if err != nil {
			return err
		}

		log.Printf("%s:\n", v.Name)
		log.Printf("  Private Key:         %x\n", vPriv)
		log.Printf("  Public Key:          %x\n", vPub)
		log.Printf("  Government Signature:%x\n\n", v.GovernmentSignature)
	}

	return nil
}

func GenerateTestVotersAndGovernmentKeyPair(numVoters int) (*ppk.KeyPair, []*voters.Voter, error) {
	govKeyPair, err := ppk.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}

	var generatedVoters []*voters.Voter
	for i := 1; i <= numVoters; i++ {
		voterKeyPair, err := ppk.GenerateKeyPair()
		if err != nil {
			return nil, nil, err
		}

		pubHash := hash.HashBytes(voterKeyPair.PublicKey.AsBytes())
		govSig, err := govKeyPair.PrivateKey.CreateSignature(pubHash)
		if err != nil {
			return nil, nil, err
		}

		v := &voters.Voter{
			Name:                fmt.Sprintf("Voter%d", i),
			KeyPair:             *voterKeyPair,
			GovernmentSignature: govSig,
		}

		generatedVoters = append(generatedVoters, v)
	}

	return govKeyPair, generatedVoters, nil
}

func CreateTestBlock(previousBlockId []byte, transactions []*models.Transaction) (*models.Block, error) {
	minerKeyPair, err := ppk.GenerateKeyPair()

	if err != nil {
		return nil, err
	}

	blockHeader := models.BlockHeader{
		Version:         1,
		PreviousBlockId: previousBlockId,
		MerkleRoot:      models.TransactionsMerkleRoot(transactions),
		Timestamp:       time.Now().Unix(),
		NBits:           uint32(difficulty.MINIMUM_DIFFICULTY),
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

func CreateTestData(numberOfBlocks int, transactionsPerBlock int) (*ppk.KeyPair, []*models.Block, map[string]*ppk.KeyPair, error) {
	govKeyPair, err := GenerateTestGovernmentKeyPair()

	if err != nil {
		return nil, nil, nil, err
	}

	genesisBlock := TestBlockRepository.GenesisBlock()
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

		err = TestBlockRepository.InsertIfNotExists(block)

		if err != nil {
			return nil, nil, nil, err
		}

		previousBlockId = block.Header.Id

		blocks[blocksCounter] = block
		blocksCounter++
	}

	return govKeyPair, blocks, keyPairs, nil
}

func GenerateTestGovernmentKeyPair() (*ppk.KeyPair, error) {
	keyPair, err := ppk.GenerateKeyPair()

	if err != nil {
		return nil, err
	}

	TestConfig.GovernmentConfig.PublicKey = keyPair.PublicKey.AsBytes()
	return keyPair, nil
}
