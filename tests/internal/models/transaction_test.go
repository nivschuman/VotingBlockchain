package models_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/models"
)

func getTestTransaction() *models.Transaction {
	transaction := models.Transaction{
		CandidateId: 1,
		Year:        2020,
		Term:        1,
	}

	return &transaction
}

func generateExpectedTransactionHash(transaction *models.Transaction) []byte {
	buf_size := 4 + 4 + 4 + len(transaction.WalletId)
	buf := make([]byte, buf_size)

	binary.BigEndian.PutUint32(buf[0:4], transaction.CandidateId)
	binary.BigEndian.PutUint32(buf[4:8], transaction.Year)
	binary.BigEndian.PutUint32(buf[8:12], transaction.Term)
	copy(buf[12:], transaction.WalletId)

	return hash.HashBytes(buf)
}

func TestGetTransactionHash(t *testing.T) {
	transaction := getTestTransaction()
	expectedHash := generateExpectedTransactionHash(transaction)
	receivedHash := transaction.GetTransactionHash()

	if !(bytes.Equal(expectedHash, receivedHash)) {
		t.Fatalf("expected hash isn't same as received hash")
	}
}

func TestTransactionSetId(t *testing.T) {
	transaction := getTestTransaction()
	expectedId := generateExpectedTransactionHash(transaction)

	transaction.SetId()

	if !(bytes.Equal(expectedId, transaction.Id)) {
		t.Fatalf("expected id isn't same as received hash")
	}
}
