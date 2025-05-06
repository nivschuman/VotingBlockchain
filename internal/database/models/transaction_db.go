package db_models

type TransactionDB struct {
	Id                  []byte `gorm:"primaryKey;column:id"`
	Version             int32  `gorm:"column:version;not null"`
	CandidateId         uint32 `gorm:"column:candidate_id;not null"`
	VoterPublicKey      []byte `gorm:"column:voter_public_key;not null"`
	GovernmentSignature []byte `gorm:"column:government_signature;not null"`
	Signature           []byte `gorm:"column:signature;not null"`
}

func (TransactionDB) TableName() string {
	return "transactions"
}
