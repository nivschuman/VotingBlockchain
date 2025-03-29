package db_models

type TransactionDB struct {
	Id          []byte `gorm:"primaryKey;column:id"`
	Version     int32  `gorm:"column:version;not null"`
	CandidateId uint32 `gorm:"column:candidate_id;not null"`
	Signature   []byte `gorm:"column:signature;not null"`

	WalletId []byte `gorm:"column:wallet_id;not null"`
	Wallet   WalletDB

	Blocks []BlockDB `gorm:"many2many:transactions_blocks;foreignKey:Id;joinForeignKey:transaction_id;References:BlockHeaderId;joinReferences:block_id"`
}

func (TransactionDB) TableName() string {
	return "transactions"
}
