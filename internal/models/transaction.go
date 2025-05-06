package models

import (
	"bytes"
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/config"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
)

type Transaction struct {
	Id                  []byte //hash of (Version, CandidateId, VoterPublicKey), 32 bytes
	Version             int32  //version of transaction, 4 bytes
	CandidateId         uint32 //id of candidate to vote for, 4 bytes
	VoterPublicKey      []byte //public key of voter marshal compressed, 33 bytes
	GovernmentSignature []byte //signature of hash of voter public key, in ASN1 format, 70-72 bytes, signed by government
	Signature           []byte //signature of Id, in ASN1 format, 70-72 bytes, signed by voter
}

func (transaction *Transaction) GetHash() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, transaction.Version)
	binary.Write(buf, binary.BigEndian, transaction.CandidateId)
	buf.Write(transaction.VoterPublicKey)

	return hash.HashBytes(buf.Bytes())
}

func (transaction *Transaction) SetId() {
	transaction.Id = transaction.GetHash()
}

func (transaction *Transaction) AsBytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, transaction.Version)
	binary.Write(buf, binary.BigEndian, transaction.CandidateId)
	buf.Write(transaction.VoterPublicKey)
	binary.Write(buf, binary.BigEndian, uint32(len(transaction.GovernmentSignature)))
	buf.Write(transaction.GovernmentSignature)
	binary.Write(buf, binary.BigEndian, uint32(len(transaction.Signature)))
	buf.Write(transaction.Signature)

	return buf.Bytes()
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

	transaction.VoterPublicKey = make([]byte, 33)
	if _, err := buf.Read(transaction.VoterPublicKey); err != nil {
		return nil, err
	}

	var governmentSignatureLength uint32
	if err := binary.Read(buf, binary.BigEndian, &governmentSignatureLength); err != nil {
		return nil, err
	}

	transaction.GovernmentSignature = make([]byte, governmentSignatureLength)
	if _, err := buf.Read(transaction.Signature); err != nil {
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

func (transaction *Transaction) SignatureIsValid() (bool, error) {
	publicKey, err := ppk.GetPublicKeyFromBytes(transaction.VoterPublicKey)

	if err != nil {
		return false, err
	}

	return publicKey.VerifySignature(transaction.Signature, transaction.GetHash()), nil
}

func (transaction *Transaction) GovernmentSignatureIsValid() (bool, error) {
	publicKey, err := ppk.GetPublicKeyFromBytes(config.GlobalConfig.GovernmentConfig.PublicKey)

	if err != nil {
		return false, err
	}

	return publicKey.VerifySignature(transaction.GovernmentSignature, hash.HashBytes(transaction.VoterPublicKey)), nil
}
