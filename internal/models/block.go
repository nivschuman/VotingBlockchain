package models

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
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
	Header  BlockHeader //header of block
	Content []Content   //array of block content
}

func (blockHeader *BlockHeader) GetHash() []byte {
	return hash.HashBytes(blockHeader.AsBytes())
}

func (blockHeader *BlockHeader) SetId() {
	blockHeader.Id = blockHeader.GetHash()
}

func (blockHeader *BlockHeader) IsHashBelowTarget() bool {
	targetBigInt := getTargetFromNBits(blockHeader.NBits)
	blockBigInt := new(big.Int).SetBytes(blockHeader.GetHash())

	return blockBigInt.Cmp(targetBigInt) <= 0
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

	for _, content := range block.Content {
		binary.Write(buf, binary.BigEndian, content.Type())

		contentBytes := content.AsBytes()
		binary.Write(buf, binary.BigEndian, uint32(len(contentBytes)))
		buf.Write(contentBytes)
	}

	return buf.Bytes()
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

	for buf.Len() > 0 {
		var contentType uint16
		if err := binary.Read(buf, binary.BigEndian, &contentType); err != nil {
			return nil, err
		}

		var contentLength uint32
		if err := binary.Read(buf, binary.BigEndian, &contentLength); err != nil {
			return nil, err
		}

		contentBytes := make([]byte, contentLength)
		if _, err := buf.Read(contentBytes); err != nil {
			return nil, err
		}

		content, err := contentFromBytes(contentType, contentBytes)

		if err != nil {
			return nil, err
		}

		block.Content = append(block.Content, content)
	}

	return block, nil
}

func contentFromBytes(contentType uint16, b []byte) (Content, error) {
	switch contentType {
	case TRANSACTION_TYPE:
		return TransactionFromBytes(b)
	case WALLET_TYPE:
		return WalletFromBytes(b)
	default:
		return nil, fmt.Errorf("unknown content type: %d", contentType)
	}
}

func getTargetFromNBits(nBits uint32) *big.Int {
	// Extract exponent (first byte)
	exponent := nBits >> 24

	// Extract coefficient (lower 3 bytes)
	coefficient := nBits & 0x00FFFFFF

	// Initialize a big integer for the target
	target := big.NewInt(int64(coefficient))

	// Shift the coefficient by (exponent - 3) to adjust the target
	if exponent > 3 {
		// Left shift by (exponent - 3) bytes, equivalent to multiplying by 256^(exponent-3)
		target.Lsh(target, uint(8*(exponent-3)))
	}

	// Return the target as a big integer
	return target
}
