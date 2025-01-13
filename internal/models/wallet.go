package models

import (
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
)

type Wallet struct {
	Id                  []byte //hash of (Year, Term, PublicKey), 32 bytes
	VoterPublicKey      []byte //public key of voter marshal compressed, 33 bytes
	Year                uint32 //year of vote that wallet is valid for, 4 bytes
	Term                uint32 //term of vote in year that wallet is valid for, 4 bytes
	GovernmentSignature []byte //signature of Id, 64 bytes, in ASN1 format, 70-72 bytes, signed by government
}

func (wallet *Wallet) GetWalletHash() []byte {
	buf_size := 4 + 4 + len(wallet.VoterPublicKey)
	buf := make([]byte, buf_size)

	binary.BigEndian.PutUint32(buf[0:4], wallet.Year)
	binary.BigEndian.PutUint32(buf[4:8], wallet.Term)
	copy(buf[8:], wallet.VoterPublicKey)

	return hash.HashBytes(buf)
}

func (wallet *Wallet) SetId() {
	wallet.Id = wallet.GetWalletHash()
}

func (wallet *Wallet) VerifySignature(signature []byte, hash []byte) (bool, error) {
	publicKey, err := ppk.GetPublicKeyFromBytes(wallet.VoterPublicKey)

	if err != nil {
		return false, err
	}

	return publicKey.VerifySignature(signature, hash), nil
}
