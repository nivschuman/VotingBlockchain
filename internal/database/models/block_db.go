package db_models

import (
	types "github.com/nivschuman/VotingBlockchain/internal/database/types"
	"github.com/nivschuman/VotingBlockchain/internal/difficulty"
)

type BlockHeaderDB struct {
	Id             []byte `gorm:"primaryKey;column:id"`
	Version        int32  `gorm:"column:version;not null"`
	MerkleRoot     []byte `gorm:"column:merkle_root;not null"`
	Timestamp      int64  `gorm:"column:timestamp;not null"`
	NBits          uint32 `gorm:"column:nbits;not null"`
	Nonce          uint32 `gorm:"column:nonce;not null"`
	MinerPublicKey []byte `gorm:"column:miner_public_key;not null"`

	PreviousBlockHeaderId *[]byte        `gorm:"column:previous_block_header_id"`
	PreviousBlockHeader   *BlockHeaderDB `gorm:"foreignKey:PreviousBlockHeaderId;references:Id;constraint:OnDelete:CASCADE"`
}

type BlockDB struct {
	Height         uint64       `gorm:"column:height:not null"`
	InActiveChain  bool         `gorm:"column:in_active_chain:not null"`
	CumulativeWork types.BigInt `gorm:"column:cumulative_work:not null"`

	BlockHeaderId []byte        `gorm:"primaryKey;column:block_header_id"`
	BlockHeader   BlockHeaderDB `gorm:"foreignKey:BlockHeaderId;references:Id"`
}

func (BlockHeaderDB) TableName() string {
	return "block_headers"
}

func (BlockDB) TableName() string {
	return "blocks"
}

func (blockHeader *BlockHeaderDB) IsHashBelowTarget() bool {
	target := difficulty.GetTargetFromNBits(blockHeader.NBits)
	return difficulty.IsHashBelowTarget(blockHeader.Id, target)
}
