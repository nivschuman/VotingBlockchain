package models

import (
	"bytes"
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

type Transaction struct {
	Id          []byte //hash of (Version, CandidateId, WalletId), 32 bytes
	Version     int32  //version of transaction, 4 bytes
	CandidateId uint32 //id of candidate to vote for, 4 bytes
	WalletId    []byte //wallet id, 32 bytes (hash)
	Signature   []byte //signature of Id, in ASN1 format, 70-72 bytes, signed by voter
}

func (transaction *Transaction) GetHash() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, transaction.Version)
	binary.Write(buf, binary.BigEndian, transaction.CandidateId)
	buf.Write(transaction.WalletId)

	return hash.HashBytes(buf.Bytes())
}

func (transaction *Transaction) SetId() {
	transaction.Id = transaction.GetHash()
}
