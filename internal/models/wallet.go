package models

import (
	"bytes"
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
)

type Wallet struct {
	Id                  []byte //hash of (Version, VoterPublicKey, ElectionId), 32 bytes
	Version             int32  //version of wallet, 4 bytes
	VoterPublicKey      []byte //public key of voter marshal compressed, 33 bytes
	ElectionId          []byte //election that wallet is valid for, 32 bytes
	GovernmentSignature []byte //signature of Id, in ASN1 format, 70-72 bytes, signed by government
}

func (wallet *Wallet) GetHash() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, wallet.Version)
	buf.Write(wallet.VoterPublicKey)
	buf.Write(wallet.ElectionId)

	return hash.HashBytes(buf.Bytes())
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

func (wallet *Wallet) AsBytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, wallet.Version)
	buf.Write(wallet.VoterPublicKey)
	buf.Write(wallet.ElectionId)
	binary.Write(buf, binary.BigEndian, uint32(len(wallet.GovernmentSignature)))
	buf.Write(wallet.GovernmentSignature)

	return buf.Bytes()
}
