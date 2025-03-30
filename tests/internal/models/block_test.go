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

	wallet, _, err := getTestWallet()

	if err != nil {
		return nil, err
	}

	transaction, err := getTestTransaction()

	if err != nil {
		return nil, err
	}

	block.Content = append(block.Content, wallet)
	block.Content = append(block.Content, transaction)

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

	if len(parsedBlock.Content) != len(block.Content) {
		t.Fatalf("have %d content but %d content was parsed", len(block.Content), len(parsedBlock.Content))
	}

	parsedWallet, ok := parsedBlock.Content[0].(*models.Wallet)

	if !ok {
		t.Fatalf("first item in parsed block isn't a wallet")
	}

	wallet, _ := block.Content[0].(*models.Wallet)

	if !bytes.Equal(wallet.Id, parsedWallet.Id) {
		t.Fatalf("bad wallet id")
	}

	parsedTransaction, ok := parsedBlock.Content[1].(*models.Transaction)

	if !ok {
		t.Fatalf("second item in parsed block isn't a transaction")
	}

	transaction, _ := block.Content[1].(*models.Transaction)

	if !bytes.Equal(transaction.Id, parsedTransaction.Id) {
		t.Fatalf("bad transaction id")
	}
}
