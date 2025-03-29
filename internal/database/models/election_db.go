package db_models

type ElectionDB struct {
	Id                  []byte `gorm:"primaryKey;column:id"`
	Version             int32  `gorm:"column:version;not null"`
	StartTimestamp      int64  `gorm:"column:start_timestamp;not null"`
	EndTimestamp        int64  `gorm:"column:end_timestamp;not null"`
	GovernmentSignature []byte `gorm:"column:government_signature;not null"`

	Wallets []WalletDB `gorm:"foreignKey:ElectionId;references:Id;constraint:OnDelete:RESTRICT,OnUpdate:RESTRICT"`
	Blocks  []BlockDB  `gorm:"many2many:elections_blocks;foreignKey:Id;joinForeignKey:election_id;References:BlockHeaderId;joinReferences:block_id"`
}

func (ElectionDB) TableName() string {
	return "elections"
}
