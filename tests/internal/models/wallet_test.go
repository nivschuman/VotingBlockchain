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
		VoterPublicKey: keyPair.PublicKey.AsBytes(),
		Year:           2020,
		Term:           1,
	}

	return wallet, keyPair, nil
}

func generateExpectedWalletHash(wallet *models.Wallet) []byte {
	buf_size := 4 + 4 + len(wallet.VoterPublicKey)
	buf := make([]byte, buf_size)

	binary.BigEndian.PutUint32(buf[0:4], wallet.Year)
	binary.BigEndian.PutUint32(buf[4:8], wallet.Term)
	copy(buf[8:], wallet.VoterPublicKey)

	return hash.HashBytes(buf)
}

func TestGetWalletHash(t *testing.T) {
	wallet, _, err := getTestWallet()

	if err != nil {
		t.Fatalf("error in getTestWallet: %v", err)
	}

	expectedHash := generateExpectedWalletHash(wallet)
	receivedHash := wallet.GetWalletHash()

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
