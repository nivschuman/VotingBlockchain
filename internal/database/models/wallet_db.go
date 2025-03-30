package db_models

type WalletDB struct {
	Id                  []byte `gorm:"primaryKey;column:id"`
	Version             int32  `gorm:"column:version;not null"`
	VoterPublicKey      []byte `gorm:"column:voter_public_key;not null"`
	GovernmentSignature []byte `gorm:"column:government_signature;not null"`

	Transactions []TransactionDB `gorm:"foreignKey:WalletId;references:Id;constraint:OnDelete:RESTRICT,OnUpdate:RESTRICT"`
	Blocks       []BlockDB       `gorm:"many2many:wallets_blocks;foreignKey:Id;joinForeignKey:wallet_id;References:BlockHeaderId;joinReferences:block_id"`
}

func (WalletDB) TableName() string {
	return "wallets"
}
