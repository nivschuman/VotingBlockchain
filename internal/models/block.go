package models

import (
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
	buf_size := 4 + 8 + 4 + 4
	buf := make([]byte, buf_size)

	binary.BigEndian.PutUint32(buf[0:4], uint32(blockHeader.Version))
	binary.BigEndian.PutUint64(buf[4:12], uint64(blockHeader.Timestamp))
	binary.BigEndian.PutUint32(buf[12:16], blockHeader.NBits)
	binary.BigEndian.PutUint32(buf[16:20], blockHeader.Nonce)

	concatenated := append(buf, blockHeader.PreviousBlockId...)
	concatenated = append(concatenated, blockHeader.MerkleRoot...)
	concatenated = append(concatenated, blockHeader.MinerPublicKey...)

	return hash.HashBytes(concatenated)
}

func (blockHeader *BlockHeader) IsHashBelowTarget() bool {
	targetBigInt := getTargetFromNBits(blockHeader.NBits)
	blockBigInt := new(big.Int).SetBytes(blockHeader.GetHash())

	return blockBigInt.Cmp(targetBigInt) <= 0
}

// version,timestamp,nBits,nonce,previousBlockId,merkleRoot,minerPublicKey
func (blockHeader *BlockHeader) AsBytes() []byte {
	buf := make([]byte, 0)

	versionBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(versionBytes, uint32(blockHeader.Version))
	buf = append(buf, versionBytes...)

	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(blockHeader.Timestamp))
	buf = append(buf, timestampBytes...)

	nBitsBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(nBitsBytes, blockHeader.NBits)
	buf = append(buf, nBitsBytes...)

	nonceBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(nonceBytes, blockHeader.Nonce)
	buf = append(buf, nonceBytes...)

	buf = append(buf, blockHeader.PreviousBlockId...)
	buf = append(buf, blockHeader.MerkleRoot...)
	buf = append(buf, blockHeader.MinerPublicKey...)

	return buf
}

func (block *Block[T]) AsBytes() []byte {
	buf := block.Header.AsBytes()

	contentTypeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(contentTypeBytes, block.ContentType)
	buf = append(buf, contentTypeBytes...)

	for _, content := range block.Content {
		buf = append(buf, content.AsBytes()...)
	}

	return buf
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
