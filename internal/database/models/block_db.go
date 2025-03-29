package db_models

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
	ContentType   uint16 `gorm:"column:content_type:not null"`
	Height        uint64 `gorm:"column:height:not null"`
	InActiveChain bool   `gorm:"column:in_active_chain:not null"`

	BlockHeaderId []byte        `gorm:"primaryKey;column:block_header_id"`
	BlockHeader   BlockHeaderDB `gorm:"foreignKey:BlockHeaderId;references:Id"`

	Transactions []TransactionDB `gorm:"many2many:transactions_blocks;foreignKey:BlockHeaderId;joinForeignKey:block_id;References:Id;joinReferences:transaction_id"`
	Elections    []ElectionDB    `gorm:"many2many:elections_blocks;foreignKey:BlockHeaderId;joinForeignKey:block_id;References:Id;joinReferences:election_id"`
	Wallets      []WalletDB      `gorm:"many2many:wallets_blocks;foreignKey:BlockHeaderId;joinForeignKey:block_id;References:Id;joinReferences:wallet_id"`
}

func (BlockHeaderDB) TableName() string {
	return "block_headers"
}

func (BlockDB) TableName() string {
	return "blocks"
}
