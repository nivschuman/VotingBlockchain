package models_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
	"github.com/nivschuman/VotingBlockchain/internal/models"
)

func getTestTransaction() (*models.Transaction, *ppk.KeyPair, error) {
	keyPair, err := ppk.GenerateKeyPair()

	if err != nil {
		return nil, nil, err
	}

	transaction := &models.Transaction{
		Version:             1,
		CandidateId:         1,
		VoterPublicKey:      keyPair.PublicKey.AsBytes(),
		GovernmentSignature: make([]byte, 72),
		Signature:           make([]byte, 72),
	}

	transaction.SetId()

	return transaction, keyPair, nil
}

func generateExpectedTransactionHash(transaction *models.Transaction) []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, transaction.Version)
	binary.Write(buf, binary.BigEndian, transaction.CandidateId)
	buf.Write(transaction.VoterPublicKey)

	return hash.HashBytes(buf.Bytes())
}

func generateExpectedTransactionBytes(transaction *models.Transaction) []byte {
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

func TestGetTransactionHash(t *testing.T) {
	transaction, _, err := getTestTransaction()

	if err != nil {
		t.Fatalf("error in getTestTransaction: %v", err)
	}

	expectedHash := generateExpectedTransactionHash(transaction)
	receivedHash := transaction.GetHash()

	if !(bytes.Equal(expectedHash, receivedHash)) {
		t.Fatalf("expected hash isn't same as received hash")
	}
}

func TestTransactionSetId(t *testing.T) {
	transaction, _, err := getTestTransaction()

	if err != nil {
		t.Fatalf("error in getTestTransaction: %v", err)
	}

	expectedId := generateExpectedTransactionHash(transaction)

	transaction.SetId()

	if !(bytes.Equal(expectedId, transaction.Id)) {
		t.Fatalf("expected id isn't same as received hash")
	}
}

func TestTransactionAsBytes(t *testing.T) {
	transaction, _, err := getTestTransaction()

	if err != nil {
		t.Fatalf("error in getTestTransaction: %v", err)
	}

	expectedBytes := generateExpectedTransactionBytes(transaction)

	if !(bytes.Equal(expectedBytes, transaction.AsBytes())) {
		t.Fatalf("expected bytes aren't the same as received bytes")
	}
}

func TestTransactionFromBytes(t *testing.T) {
	transaction, _, err := getTestTransaction()

	if err != nil {
		t.Fatalf("error in getTestTransaction: %v", err)
	}

	transactionBytes := transaction.AsBytes()
	parsedTransaction, err := models.TransactionFromBytes(transactionBytes)

	if err != nil {
		t.Fatalf("error in transaction from bytes: %v", err)
	}

	if !bytes.Equal(parsedTransaction.Id, transaction.Id) {
		t.Fatalf("bad id for parsed transaction")
	}
}

func TestIsValidSignature_WhenSignatureIsValid(t *testing.T) {
	transaction, keyPair, err := getTestTransaction()

	if err != nil {
		t.Fatalf("error in getTestTransaction: %v", err)
	}

	transaction.Signature, err = keyPair.PrivateKey.CreateSignature(transaction.Id)

	if err != nil {
		t.Fatalf("error with create signature: %v", err)
	}

	valid, err := transaction.IsValidSignature()

	if err != nil {
		t.Fatalf("error with is valid signature: %v", err)
	}

	if !valid {
		t.Fatalf("verify signature returned false")
	}
}

func TestIsValidSignature_WhenSignatureIsInvalid(t *testing.T) {
	transaction, keyPair, err := getTestTransaction()

	if err != nil {
		t.Fatalf("error in getTestTransaction: %v", err)
	}

	h := hash.HashString("test")
	transaction.Signature, err = keyPair.PrivateKey.CreateSignature(h)

	if err != nil {
		t.Fatalf("error with create signature: %v", err)
	}

	valid, err := transaction.IsValidSignature()

	if err != nil {
		t.Fatalf("error with is valid signature: %v", err)
	}

	if valid {
		t.Fatalf("verify signature returned true")
	}
}
