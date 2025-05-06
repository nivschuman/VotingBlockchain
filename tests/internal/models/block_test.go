package models_test

import (
	"bytes"
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/models"
)

func getTestBlockHeader() models.BlockHeader {
	blockHeader := models.BlockHeader{
		Version:         1,
		PreviousBlockId: make([]byte, 32),
		MerkleRoot:      make([]byte, 32),
		Timestamp:       0,
		NBits:           uint32(0x1d00ffff),
		Nonce:           0,
		MinerPublicKey:  make([]byte, 33),
	}

	blockHeader.SetId()

	return blockHeader
}

func getTestBlock() (*models.Block, error) {
	blockHeader := getTestBlockHeader()
	blockHeader.SetId()

	block := &models.Block{
		Header: blockHeader,
	}

	transaction, _, err := getTestTransaction()

	if err != nil {
		return nil, err
	}

	block.Transactions = append(block.Transactions, transaction)

	return block, nil
}

func TestBlockHeaderFromBytes(t *testing.T) {
	blockHeader := getTestBlockHeader()

	blockHeaderBytes := blockHeader.AsBytes()
	parsedBlockHeader, err := models.BlockHeaderFromBytes(blockHeaderBytes)

	if err != nil {
		t.Fatalf("error in block header from bytes: %v", err)
	}

	if !bytes.Equal(parsedBlockHeader.Id, blockHeader.Id) {
		t.Fatalf("bad id for parsed block header")
	}
}

func TestBlockFromBytes(t *testing.T) {
	block, err := getTestBlock()

	if err != nil {
		t.Fatalf("error in get test block: %v", err)
	}

	blockBytes := block.AsBytes()
	parsedBlock, err := models.BlockFromBytes(blockBytes)

	if err != nil {
		t.Fatalf("error in block from bytes: %v", err)
	}

	if !bytes.Equal(parsedBlock.Header.Id, block.Header.Id) {
		t.Fatalf("bad id for parsed block")
	}

	if len(parsedBlock.Transactions) != len(block.Transactions) {
		t.Fatalf("have %d transactions but %d transactions were parsed", len(block.Transactions), len(parsedBlock.Transactions))
	}

	parsedTransaction := parsedBlock.Transactions[0]
	transaction := block.Transactions[0]

	if !bytes.Equal(transaction.Id, parsedTransaction.Id) {
		t.Fatalf("bad transaction id")
	}
}
