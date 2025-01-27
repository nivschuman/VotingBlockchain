package models_test

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/models"
)

func getTestElection() *models.Election {
	currentTime := time.Now()
	startTimestamp := currentTime.Unix()
	endTimestamp := currentTime.Add(7 * 24 * time.Hour).Unix()

	election := models.Election{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}

	election.SetId()

	return &election
}

func generateExpectedElectionHash(election *models.Election) []byte {
	buf_size := 8 + 8
	buf := make([]byte, buf_size)

	binary.BigEndian.PutUint64(buf[0:8], uint64(election.StartTimestamp))
	binary.BigEndian.PutUint64(buf[8:16], uint64(election.EndTimestamp))

	return hash.HashBytes(buf)
}

func TestElectionGetHash(t *testing.T) {
	election := getTestElection()
	expectedHash := generateExpectedElectionHash(election)
	receivedHash := election.GetHash()

	if !(bytes.Equal(expectedHash, receivedHash)) {
		t.Fatalf("expected hash isn't same as received hash")
	}
}

func TestElectionSetId(t *testing.T) {
	election := getTestElection()
	expectedId := generateExpectedElectionHash(election)

	election.SetId()

	if !(bytes.Equal(expectedId, election.Id)) {
		t.Fatalf("expected id isn't same as received hash")
	}
}
