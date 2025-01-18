package models

import (
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

type Transaction struct {
	Id          []byte //hash of (CandidateId, ElectionId, WalletId), 32 bytes
	CandidateId uint32 //id of candidate to vote for, 4 bytes
	ElectionId  []byte //id of election to vote in, 32 bytes (hash)
	WalletId    []byte //wallet id, 32 bytes (hash)
	Signature   []byte //signature of Id, 64 bytes, in ASN1 format, 70-72 bytes, signed by voter
}

func (transaction *Transaction) GetHash() []byte {
	buf_size := 4
	buf := make([]byte, buf_size)
	binary.BigEndian.PutUint32(buf, transaction.CandidateId)

	concatenated := append(buf, transaction.ElectionId...)
	concatenated = append(concatenated, transaction.WalletId...)

	return hash.HashBytes(concatenated)
}

func (transaction *Transaction) SetId() {
	transaction.Id = transaction.GetHash()
}
