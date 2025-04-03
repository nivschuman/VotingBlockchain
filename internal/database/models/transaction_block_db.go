package db_models

type TransactionBlockDB struct {
	BlockHeaderId []byte `gorm:"primaryKey;column:block_header_id;not null"`
	TransactionId []byte `gorm:"primaryKey;column:transaction_id;not null"`
	Order         uint32 `gorm:"column:order;not null"`

	Block       BlockDB       `gorm:"foreignKey:BlockHeaderId;references:BlockHeaderId;constraint:OnDelete:RESTRICT"`
	Transaction TransactionDB `gorm:"foreignKey:TransactionId;references:Id;constraint:OnDelete:RESTRICT"`
}

func (TransactionBlockDB) TableName() string {
	return "transactions_blocks"
}
