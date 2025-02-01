package models_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/models"
)

func getTestTransaction() (*models.Transaction, error) {
	wallet, _, err := getTestWallet()

	if err != nil {
		return nil, err
	}

	transaction := models.Transaction{
		CandidateId: 1,
		WalletId:    wallet.Id,
	}

	return &transaction, nil
}

func generateExpectedTransactionHash(transaction *models.Transaction) []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, transaction.Version)
	binary.Write(buf, binary.BigEndian, transaction.CandidateId)
	buf.Write(transaction.WalletId)

	return hash.HashBytes(buf.Bytes())
}

func generateExpectedTransactionBytes(transaction *models.Transaction) []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, transaction.Version)
	binary.Write(buf, binary.BigEndian, transaction.CandidateId)
	buf.Write(transaction.WalletId)
	binary.Write(buf, binary.BigEndian, uint32(len(transaction.Signature)))
	buf.Write(transaction.Signature)

	return buf.Bytes()
}

func TestGetTransactionHash(t *testing.T) {
	transaction, err := getTestTransaction()

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
	transaction, err := getTestTransaction()

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
	transaction, err := getTestTransaction()

	if err != nil {
		t.Fatalf("error in getTestTransaction: %v", err)
	}

	expectedBytes := generateExpectedTransactionBytes(transaction)

	if !(bytes.Equal(expectedBytes, transaction.AsBytes())) {
		t.Fatalf("expected bytes aren't the same as received bytes")
	}
}
