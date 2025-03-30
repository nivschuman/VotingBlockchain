package models

import (
	"bytes"
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

const TRANSACTION_TYPE = uint16(1)

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

func (transaction *Transaction) AsBytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, transaction.Version)
	binary.Write(buf, binary.BigEndian, transaction.CandidateId)
	buf.Write(transaction.WalletId)
	binary.Write(buf, binary.BigEndian, uint32(len(transaction.Signature)))
	buf.Write(transaction.Signature)

	return buf.Bytes()
}

func (transaction *Transaction) Type() uint16 {
	return TRANSACTION_TYPE
}

func TransactionFromBytes(b []byte) (*Transaction, error) {
	buf := bytes.NewReader(b)

	transaction := &Transaction{}

	if err := binary.Read(buf, binary.BigEndian, &transaction.Version); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.BigEndian, &transaction.CandidateId); err != nil {
		return nil, err
	}

	transaction.WalletId = make([]byte, 32)
	if _, err := buf.Read(transaction.WalletId); err != nil {
		return nil, err
	}

	var signatureLength uint32
	if err := binary.Read(buf, binary.BigEndian, &signatureLength); err != nil {
		return nil, err
	}

	transaction.Signature = make([]byte, signatureLength)
	if _, err := buf.Read(transaction.Signature); err != nil {
		return nil, err
	}

	transaction.SetId()

	return transaction, nil
}
