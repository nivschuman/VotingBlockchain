package models

import (
	"bytes"
	"encoding/binary"
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

type Block[T Content] struct {
	Header      BlockHeader //header of block
	ContentType uint16      //code of type of content in block
	Content     []T         //array of block content
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

func (block *Block[T]) AsBytes() []byte {
	buf := new(bytes.Buffer)

	buf.Write(block.Header.AsBytes())
	binary.Write(buf, binary.BigEndian, uint16(block.ContentType))

	for _, content := range block.Content {
		buf.Write(content.AsBytes())
	}

	return buf.Bytes()
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
