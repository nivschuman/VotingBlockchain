package models_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
	"github.com/nivschuman/VotingBlockchain/internal/models"
)

func getTestWallet() (*models.Wallet, *ppk.KeyPair, error) {
	keyPair, err := ppk.GenerateKeyPair()

	if err != nil {
		return nil, nil, err
	}

	wallet := &models.Wallet{
		Version:             1,
		VoterPublicKey:      keyPair.PublicKey.AsBytes(),
		GovernmentSignature: make([]byte, 72),
	}

	wallet.SetId()

	return wallet, keyPair, nil
}

func generateExpectedWalletHash(wallet *models.Wallet) []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, wallet.Version)
	buf.Write(wallet.VoterPublicKey)

	return hash.HashBytes(buf.Bytes())
}

func generateExpectedWalletBytes(wallet *models.Wallet) []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, wallet.Version)
	buf.Write(wallet.VoterPublicKey)
	binary.Write(buf, binary.BigEndian, uint32(len(wallet.GovernmentSignature)))
	buf.Write(wallet.GovernmentSignature)

	return buf.Bytes()
}

func TestGetWalletHash(t *testing.T) {
	wallet, _, err := getTestWallet()

	if err != nil {
		t.Fatalf("error in getTestWallet: %v", err)
	}

	expectedHash := generateExpectedWalletHash(wallet)
	receivedHash := wallet.GetHash()

	if !(bytes.Equal(expectedHash, receivedHash)) {
		t.Fatalf("expected hash isn't same as received hash")
	}
}

func TestWalletSetId(t *testing.T) {
	wallet, _, err := getTestWallet()

	if err != nil {
		t.Fatalf("error in getTestWallet: %v", err)
	}

	expectedId := generateExpectedWalletHash(wallet)
	wallet.SetId()

	if !(bytes.Equal(expectedId, wallet.Id)) {
		t.Fatalf("expected hash isn't same as received hash")
	}
}

func TestWalletAsBytes(t *testing.T) {
	wallet, _, err := getTestWallet()

	if err != nil {
		t.Fatalf("error in getTestWallet: %v", err)
	}

	expectedBytes := generateExpectedWalletBytes(wallet)

	if !(bytes.Equal(expectedBytes, wallet.AsBytes())) {
		t.Fatalf("expected hash isn't same as received hash")
	}
}

func TestWalletVerifySignature_WhenSignatureIsValid(t *testing.T) {
	wallet, keyPair, err1 := getTestWallet()

	if err1 != nil {
		t.Fatalf("error in getTestWallet: %v", err1)
	}

	h := hash.HashString("test")
	signature, err2 := keyPair.PrivateKey.CreateSignature(h)

	if err2 != nil {
		t.Fatalf("error with create signature: %v", err2)
	}

	valid, err3 := wallet.VerifySignature(signature, h)

	if err3 != nil {
		t.Fatalf("error with verify signature: %v", err3)
	}

	if !valid {
		t.Fatalf("verify signature returned false")
	}
}

func TestWalletVerifySignature_WhenSignatureIsInvalid(t *testing.T) {
	wallet, keyPair, err1 := getTestWallet()

	if err1 != nil {
		t.Fatalf("error in getTestWallet: %v", err1)
	}

	h := hash.HashString("test")
	signature, err2 := keyPair.PrivateKey.CreateSignature(h)

	if err2 != nil {
		t.Fatalf("error with create signature: %v", err2)
	}

	h = hash.HashString("other")
	valid, err3 := wallet.VerifySignature(signature, h)

	if err3 != nil {
		t.Fatalf("error with verify signature: %v", err3)
	}

	if valid {
		t.Fatalf("verify signature returned true")
	}
}

func TestWalletFromBytes(t *testing.T) {
	wallet, _, err := getTestWallet()

	if err != nil {
		t.Fatalf("error in getTestWallet: %v", err)
	}

	walletBytes := wallet.AsBytes()
	parsedWallet, err := models.WalletFromBytes(walletBytes)

	if err != nil {
		t.Fatalf("error in wallet from bytes: %v", err)
	}

	if !bytes.Equal(wallet.Id, parsedWallet.Id) {
		t.Fatalf("bad id for parsed wallet")
	}
}
