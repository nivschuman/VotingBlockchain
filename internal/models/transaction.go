package models

import (
	"fmt"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
)

type Transaction struct {
	Id             []byte //hash of (CandidateId, Year, VoterPublicKey), 32 bytes
	CandidateId    uint32 //id of candidate to vote for, 4 bytes
	Year           uint32 //year of vote, 4 bytes
	VoterPublicKey []byte //marshal compressed ECDSA public key, 33 bytes
	Signature      []byte //signature of Id, 64 bytes, in ASN1 format, 70-72 bytes
}

func (transaction *Transaction) GetTransactionHash() []byte {
	data := fmt.Sprintf("%d%d%s", transaction.CandidateId, transaction.Year, transaction.VoterPublicKey)
	return hash.HashStringAsBytes(data)
}

func (transaction *Transaction) SetId() {
	transaction.Id = transaction.GetTransactionHash()
}

func (transaction *Transaction) IsValidSignature() (bool, error) {
	hash := transaction.GetTransactionHash()
	publicKey, err := ppk.GetPublicKeyFromBytes(transaction.VoterPublicKey)

	if err != nil {
		return false, err
	}

	return publicKey.VerifySignature(transaction.Signature, hash), nil
}
