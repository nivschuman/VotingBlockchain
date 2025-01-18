package models

import (
	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
)

type Wallet struct {
	Id                  []byte //hash of (Year, Term, PublicKey), 32 bytes
	VoterPublicKey      []byte //public key of voter marshal compressed, 33 bytes
	ElectionId          []byte //election that wallet is valid for, 32 bytes
	GovernmentSignature []byte //signature of Id, 64 bytes, in ASN1 format, 70-72 bytes, signed by government
}

func (wallet *Wallet) GetHash() []byte {
	concatenated := append(wallet.VoterPublicKey, wallet.ElectionId...)

	return hash.HashBytes(concatenated)
}

func (wallet *Wallet) SetId() {
	wallet.Id = wallet.GetHash()
}

func (wallet *Wallet) VerifySignature(signature []byte, hash []byte) (bool, error) {
	publicKey, err := ppk.GetPublicKeyFromBytes(wallet.VoterPublicKey)

	if err != nil {
		return false, err
	}

	return publicKey.VerifySignature(signature, hash), nil
}
