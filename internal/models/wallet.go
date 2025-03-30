package models

import (
	"bytes"
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
)

const WALLET_TYPE = uint16(2)

type Wallet struct {
	Id                  []byte //hash of (Version, VoterPublicKey), 32 bytes
	Version             int32  //version of wallet, 4 bytes
	VoterPublicKey      []byte //public key of voter marshal compressed, 33 bytes
	GovernmentSignature []byte //signature of Id, in ASN1 format, 70-72 bytes, signed by government
}

func (wallet *Wallet) GetHash() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, wallet.Version)
	buf.Write(wallet.VoterPublicKey)

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
	binary.Write(buf, binary.BigEndian, uint32(len(wallet.GovernmentSignature)))
	buf.Write(wallet.GovernmentSignature)

	return buf.Bytes()
}

func (wallet *Wallet) Type() uint16 {
	return WALLET_TYPE
}

func WalletFromBytes(b []byte) (*Wallet, error) {
	buf := bytes.NewReader(b)

	wallet := &Wallet{}

	if err := binary.Read(buf, binary.BigEndian, &wallet.Version); err != nil {
		return nil, err
	}

	wallet.VoterPublicKey = make([]byte, 33)
	if _, err := buf.Read(wallet.VoterPublicKey); err != nil {
		return nil, err
	}

	var signatureLength uint32
	if err := binary.Read(buf, binary.BigEndian, &signatureLength); err != nil {
		return nil, err
	}

	wallet.GovernmentSignature = make([]byte, signatureLength)
	if _, err := buf.Read(wallet.GovernmentSignature); err != nil {
		return nil, err
	}

	wallet.SetId()

	return wallet, nil
}
