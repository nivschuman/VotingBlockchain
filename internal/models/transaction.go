package models

import (
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

type Transaction struct {
	Id          []byte //hash of (CandidateId, Year, Term, WalletId), 32 bytes
	CandidateId uint32 //id of candidate to vote for, 4 bytes
	Year        uint32 //year of vote, 4 bytes
	Term        uint32 //term of vote in year, 4 bytes
	WalletId    []byte //wallet id, 32 bytes (hash)
	Signature   []byte //signature of Id, 64 bytes, in ASN1 format, 70-72 bytes
}

func (transaction *Transaction) GetTransactionHash() []byte {
	buf_size := 4 + 4 + 4 + len(transaction.WalletId)
	buf := make([]byte, buf_size)

	binary.BigEndian.PutUint32(buf[0:4], transaction.CandidateId)
	binary.BigEndian.PutUint32(buf[4:8], transaction.Year)
	binary.BigEndian.PutUint32(buf[8:12], transaction.Term)
	copy(buf[12:], transaction.WalletId)

	return hash.HashBytes(buf)
}

func (transaction *Transaction) SetId() {
	transaction.Id = transaction.GetTransactionHash()
}
