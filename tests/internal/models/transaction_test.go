package transaction_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
	"github.com/nivschuman/VotingBlockchain/internal/models/transaction"
)

func setupTest() (*transaction.Transaction, *ppk.KeyPair, error) {
	keyPair, err1 := ppk.GenerateKeyPair()

	if err1 != nil {
		return nil, nil, err1
	}

	publicKeyBytes, err2 := keyPair.PublicKey.AsBytes()

	if err2 != nil {
		return nil, nil, err2
	}

	transaction := transaction.Transaction{
		CandidateId:    1,
		Year:           2020,
		VoterPublicKey: publicKeyBytes,
	}

	return &transaction, keyPair, nil
}

func TestGetTransactionHash(t *testing.T) {
	transaction, _, err := setupTest()

	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	data := fmt.Sprintf("%d%d%s", transaction.CandidateId, transaction.Year, transaction.VoterPublicKey)
	expectedHash := hash.HashStringAsBytes(data)
	receivedHash := transaction.GetTransactionHash()

	if !(bytes.Equal(expectedHash, receivedHash)) {
		t.Fatalf("expected hash isn't same as received hash")
	}
}

func TestSetId(t *testing.T) {
	transaction, _, err := setupTest()

	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	data := fmt.Sprintf("%d%d%s", transaction.CandidateId, transaction.Year, transaction.VoterPublicKey)
	expectedId := hash.HashStringAsBytes(data)

	transaction.SetId()

	if !(bytes.Equal(expectedId, transaction.Id)) {
		t.Fatalf("expected id isn't same as received hash")
	}
}

func TestIsValidSignature_WhenSignatureIsValid(t *testing.T) {
	transaction, keyPair, err1 := setupTest()

	if err1 != nil {
		t.Fatalf("setup failed: %v", err1)
	}

	hash := transaction.GetTransactionHash()
	signature, err2 := keyPair.PrivateKey.CreateSignature(hash)

	if err2 != nil {
		t.Fatalf("creating signature failed: %v", err1)
	}

	transaction.Signature = signature

	if !transaction.IsValidSignature() {
		t.Fatalf("IsValidSignature returned false")
	}
}
