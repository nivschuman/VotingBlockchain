package models

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/difficulty"
)

type BlockHeader struct {
	Id              []byte //hash of (Version, Timestamp, NBits, Nonce, PreviousBlockId, MerkleRoot, MinerPublicKey), 32 bytes
	Version         int32  //version of block, 4 bytes
	PreviousBlockId []byte //hash of previous block, 32 bytes
	MerkleRoot      []byte //merkle root of merkle tree of block content, 32 bytes
	Timestamp       int64  //unix timestamp of when the miner started hashing the header, 8 bytes
	NBits           uint32 //encoded version of target threshold this blocks header has must be less than or equal to, 4 bytes
	Nonce           uint32 //arbitrary numbers miners change in order to produce hash less than or equal to the target threshold, 4 bytes
	MinerPublicKey  []byte //public key of miner that made the block, marshal compressed, 33 bytes
}

type Block struct {
	Header       BlockHeader    //header of block
	Transactions []*Transaction //transactions inside block (ordered)
}

func (blockHeader *BlockHeader) GetHash() []byte {
	return hash.HashBytes(blockHeader.AsBytes())
}

func (blockHeader *BlockHeader) SetId() {
	blockHeader.Id = blockHeader.GetHash()
}

func (blockHeader *BlockHeader) IsHashBelowTarget() bool {
	target := difficulty.GetTargetFromNBits(blockHeader.NBits)
	return difficulty.IsHashBelowTarget(blockHeader.GetHash(), target)
}

func (blockHeader *BlockHeader) AsBytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, blockHeader.Version)
	binary.Write(buf, binary.BigEndian, blockHeader.Timestamp)
	binary.Write(buf, binary.BigEndian, blockHeader.NBits)
	binary.Write(buf, binary.BigEndian, blockHeader.Nonce)
	buf.Write(blockHeader.PreviousBlockId)
	buf.Write(blockHeader.MerkleRoot)
	buf.Write(blockHeader.MinerPublicKey)

	return buf.Bytes()
}

func BlockHeaderFromBytes(b []byte) (*BlockHeader, error) {
	buf := bytes.NewReader(b)
	blockHeader := &BlockHeader{}

	if err := binary.Read(buf, binary.BigEndian, &blockHeader.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &blockHeader.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &blockHeader.NBits); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &blockHeader.Nonce); err != nil {
		return nil, err
	}

	blockHeader.PreviousBlockId = make([]byte, 32)
	if _, err := buf.Read(blockHeader.PreviousBlockId); err != nil {
		return nil, err
	}

	blockHeader.MerkleRoot = make([]byte, 32)
	if _, err := buf.Read(blockHeader.MerkleRoot); err != nil {
		return nil, err
	}

	blockHeader.MinerPublicKey = make([]byte, 33)
	if _, err := buf.Read(blockHeader.MinerPublicKey); err != nil {
		return nil, err
	}

	blockHeader.SetId()

	return blockHeader, nil
}

func (block *Block) AsBytes() []byte {
	buf := new(bytes.Buffer)

	buf.Write(block.Header.AsBytes())

	binary.Write(buf, binary.BigEndian, uint32(len(block.Transactions)))
	for _, tx := range block.Transactions {
		txBytes := tx.AsBytes()
		binary.Write(buf, binary.BigEndian, uint32(len(txBytes)))
		buf.Write(txBytes)
	}

	return buf.Bytes()
}

func (block *Block) GetBlockWork() *big.Int {
	return difficulty.CalculateWork(block.Header.NBits)
}

func BlockFromBytes(b []byte) (*Block, error) {
	buf := bytes.NewReader(b)
	block := &Block{}

	headerBytes := make([]byte, 117)
	if _, err := buf.Read(headerBytes); err != nil {
		return nil, err
	}

	blockHeader, err := BlockHeaderFromBytes(headerBytes)
	if err != nil {
		return nil, err
	}
	block.Header = *blockHeader

	var numTransactions uint32
	if err := binary.Read(buf, binary.BigEndian, &numTransactions); err != nil {
		return nil, err
	}

	for i := uint32(0); i < numTransactions; i++ {
		var txLength uint32
		if err := binary.Read(buf, binary.BigEndian, &txLength); err != nil {
			return nil, err
		}

		txBytes := make([]byte, txLength)
		if _, err := buf.Read(txBytes); err != nil {
			return nil, err
		}

		tx, err := TransactionFromBytes(txBytes)
		if err != nil {
			return nil, err
		}

		block.Transactions = append(block.Transactions, tx)
	}

	return block, nil
}
